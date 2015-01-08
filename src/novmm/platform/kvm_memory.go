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
const int IoctlSetUserMemoryRegion = KVM_SET_USER_MEMORY_REGION;
const int IoctlTranslate = KVM_TRANSLATE;

// IOCTL flags.
const int IoctlFlagMemLogDirtyPages = KVM_MEM_LOG_DIRTY_PAGES;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

func (vm *Vm) MapUserMemory(
	start Paddr,
	size uint64,
	mmap []byte) error {

	// See NOTE above about read-only memory.
	// As we will not support it for the moment,
	// we do not expose it through the interface.
	// Leveraging that feature will likely require
	// a small amount of re-architecting in any case.
	var region C.struct_kvm_userspace_memory_region
	region.slot = C.__u32(vm.mem_region)
	region.flags = C.__u32(0)
	region.guest_phys_addr = C.__u64(start)
	region.memory_size = C.__u64(size)
	region.userspace_addr = C.__u64(uintptr(unsafe.Pointer(&mmap[0])))

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlSetUserMemoryRegion),
		uintptr(unsafe.Pointer(&region)))
	if e != 0 {
		return e
	}

	// We're set, bump our slot.
	vm.mem_region += 1
	return nil
}

func (vm *Vm) MapReservedMemory(
	start Paddr,
	size uint64) error {

	// Nothing to do.
	return nil
}

func (vcpu *Vcpu) Translate(
	vaddr Vaddr) (Paddr, bool, bool, bool, error) {

	// Perform the translation.
	var translation C.struct_kvm_translation
	translation.linear_address = C.__u64(vaddr)
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlTranslate),
		uintptr(unsafe.Pointer(&translation)))
	if e != 0 {
		return Paddr(0), false, false, false, e
	}

	paddr := Paddr(translation.physical_address)
	valid := translation.valid != C.__u8(0)
	writeable := translation.writeable != C.__u8(0)
	usermode := translation.valid != C.__u8(0)

	return paddr, valid, writeable, usermode, nil
}
