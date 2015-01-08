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

package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int IoctlGetClock = KVM_GET_CLOCK;
const int IoctlSetClock = KVM_SET_CLOCK;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// Our clock state.
//
type Clock struct {
	Time  uint64 `json:"time"`
	Flags uint32 `json:"flags"`
}

func (vm *Vm) GetClock() (Clock, error) {

	// Execute the ioctl.
	var kvm_clock_data C.struct_kvm_clock_data
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlGetClock),
		uintptr(unsafe.Pointer(&kvm_clock_data)))
	if e != 0 {
		return Clock{}, e
	}

	return Clock{
		Time:  uint64(kvm_clock_data.clock),
		Flags: uint32(kvm_clock_data.flags),
	}, nil
}

func (vm *Vm) SetClock(clock Clock) error {

	// Execute the ioctl.
	var kvm_clock_data C.struct_kvm_clock_data
	kvm_clock_data.clock = C.__u64(clock.Time)
	kvm_clock_data.flags = C.__u32(clock.Flags)
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlSetClock),
		uintptr(unsafe.Pointer(&kvm_clock_data)))
	if e != 0 {
		return e
	}

	return nil
}
