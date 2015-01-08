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
const int IoctlGetApiVersion = KVM_GET_API_VERSION;
const int IoctlCreateVm = KVM_CREATE_VM;
const int IoctlGetVcpuMmapSize = KVM_GET_VCPU_MMAP_SIZE;
*/
import "C"

import (
	"syscall"
)

type Vm struct {
	// The VM fd.
	fd int

	// The next vcpu id to create.
	next_id int32

	// The next memory region slot to create.
	// This is not serialized because we will
	// recreate all regions (and the ordering
	// may even be different the 2nd time round).
	mem_region int

	// Our cpuid data.
	// At the moment, we just expose the full
	// host flags to the guest.
	cpuid []Cpuid

	// Our MSRs.
	msrs []uint32

	// The size of the vcpu mmap structs.
	mmap_size int

	// Our vcpus.
	vcpus []*Vcpu
}

func getMmapSize(fd int) (int, error) {
	// Get the size of the Mmap structure.
	r, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(C.IoctlGetVcpuMmapSize),
		0)
	if e != 0 {
		return 0, e
	}
	return int(r), nil
}

func NewVm() (*Vm, error) {
	fd, err := syscall.Open("/dev/kvm", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)

	// Check API version.
	version, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(C.IoctlGetApiVersion),
		0)
	if version != 12 || e != 0 {
		return nil, e
	}

	// Check our extensions.
	for _, capSpec := range requiredCapabilities {
		err = checkCapability(fd, capSpec)
		if err != nil {
			return nil, err
		}
	}

	// Make sure we have the mmap size.
	mmap_size, err := getMmapSize(fd)
	if err != nil {
		return nil, err
	}

	// Make sure we have cpuid data.
	cpuid, err := defaultCpuid(fd)
	if err != nil {
		return nil, err
	}

	// Get our list of available MSRs.
	msrs, err := availableMsrs(fd)
	if err != nil {
		return nil, err
	}

	// Create new VM.
	vmfd, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(C.IoctlCreateVm),
		0)
	if e != 0 {
		return nil, e
	}

	// Make sure this VM gets closed.
	// (Same thing is done for Vcpus).
	syscall.CloseOnExec(int(vmfd))

	// Prepare our VM object.
	vm := &Vm{
		fd:        int(vmfd),
		vcpus:     make([]*Vcpu, 0, 0),
		cpuid:     cpuid,
		msrs:      msrs,
		mmap_size: mmap_size,
	}

	return vm, nil
}

func (vm *Vm) Dispose() error {

	for _, vcpu := range vm.vcpus {
		vcpu.Dispose()
	}

	return syscall.Close(vm.fd)
}

func (vm *Vm) Vcpus() []*Vcpu {
	return vm.vcpus
}

func (vm *Vm) VcpuInfo() ([]VcpuInfo, error) {

	err := vm.Pause(false)
	if err != nil {
		return nil, err
	}
	defer vm.Unpause(false)

	vcpus := make([]VcpuInfo, 0, len(vm.vcpus))
	for _, vcpu := range vm.vcpus {
		vcpuinfo, err := NewVcpuInfo(vcpu)
		if err != nil {
			return nil, err
		}

		vcpus = append(vcpus, vcpuinfo)
	}

	return vcpus, nil
}

func (vm *Vm) Pause(manual bool) error {

	// Pause all vcpus.
	for i, vcpu := range vm.vcpus {
		err := vcpu.Pause(manual)
		if err != nil && err != AlreadyPaused {
			// Rollback.
			// NOTE: Start with the previous.
			for i -= 1; i >= 0; i -= 1 {
				vm.vcpus[i].Unpause(manual)
			}
			return err
		}
	}

	// Done.
	return nil
}

func (vm *Vm) Unpause(manual bool) error {

	// Unpause all vcpus.
	for i, vcpu := range vm.vcpus {
		err := vcpu.Unpause(manual)
		if err != nil && err != NotPaused {
			// Rollback.
			// NOTE: Start with the previous.
			for i -= 1; i >= 0; i -= 1 {
				vm.vcpus[i].Pause(manual)
			}
			return err
		}
	}

	// Done.
	return nil
}
