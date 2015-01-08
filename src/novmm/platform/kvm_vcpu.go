// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux
package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int IoctlCreateVcpu = KVM_CREATE_VCPU;
const int IoctlSetGuestDebug = KVM_SET_GUEST_DEBUG;

// IOCTL flags.
const int IoctlGuestDebugEnable = KVM_GUESTDBG_ENABLE|KVM_GUESTDBG_SINGLESTEP;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

type Vcpu struct {
	// The VCPU id.
	Id uint

	// The VCPU fd.
	fd int

	// The mmap-structure.
	// NOTE: mmap is the go pointer to the bytes,
	// kvm points to same data but is interpreted.
	mmap []byte
	kvm  *C.struct_kvm_run

	// Cached registers.
	// See data.go for the serialization code.
	regs  C.struct_kvm_regs
	sregs C.struct_kvm_sregs

	// Caching parameters.
	regs_cached  bool
	sregs_cached bool
	regs_dirty   bool
	sregs_dirty  bool

	// Our available MSRs.
	msrs []uint32

	// Our default cpuid.
	cpuid []Cpuid

	// Is this stepping?
	is_stepping bool

	// Our run information.
	RunInfo
}

func (vm *Vm) NewVcpu(id uint) (*Vcpu, error) {

	// Create a new Vcpu.
	vcpufd, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlCreateVcpu),
		uintptr(id))
	if e != 0 {
		return nil, e
	}

	// Make sure this disappears on exec.
	// This is a race condition here, but realistically
	// we are not going to be restarting at this point.
	syscall.CloseOnExec(int(vcpufd))

	// Map our shared data.
	mmap, err := syscall.Mmap(
		int(vcpufd),
		0,
		vm.mmap_size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)
	if err != nil {
		syscall.Close(int(vcpufd))
		return nil, err
	}
	kvm_run := (*C.struct_kvm_run)(unsafe.Pointer(&mmap[0]))

	// Add our Vcpu.
	vcpu := &Vcpu{
		Id:    id,
		fd:    int(vcpufd),
		mmap:  mmap,
		kvm:   kvm_run,
		msrs:  vm.msrs,
		cpuid: vm.cpuid,
	}
	vm.vcpus = append(vm.vcpus, vcpu)

	// Set our default cpuid.
	// (This may be overriden later).
	err = vcpu.SetCpuid(vm.cpuid)
	if err != nil {
		vcpu.Dispose()
		return nil, err
	}

	// Return our VCPU object.
	return vcpu, vcpu.initRunInfo()
}

func (vcpu *Vcpu) Dispose() error {

	// Halt the processor.
	err := vcpu.SetMpState(MpStateHalted)
	if err != nil {
		return err
	}

	// Cleanup our resources.
	syscall.Munmap(vcpu.mmap)
	return syscall.Close(vcpu.fd)
}

func (vcpu *Vcpu) SetStepping(step bool) error {

	var guest_debug C.struct_kvm_guest_debug

	if step == vcpu.is_stepping {
		// Already set.
		return nil
	} else if step {
		guest_debug.control = C.__u32(C.IoctlGuestDebugEnable)
	} else {
		guest_debug.control = 0
	}

	// Execute our debug ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetGuestDebug),
		uintptr(unsafe.Pointer(&guest_debug)))
	if e != 0 {
		return e
	}

	// We're okay.
	vcpu.is_stepping = step
	return nil
}

func (vcpu *Vcpu) IsStepping() bool {
	return vcpu.is_stepping
}
