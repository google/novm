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
const int IoctlCreatePit2 = KVM_CREATE_PIT2;
const int IoctlGetPit2 = KVM_GET_PIT2;
const int IoctlSetPit2 = KVM_SET_PIT2;

// Size of pit state.
const int PitSize = sizeof(struct kvm_pit_state2);
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// PitState --
//
// We represent the PitState as a blob.
// This representation should be relatively
// safe from a forward-compatibility perspective,
// as KVM internally will take care of reserving
// bits and ensuring compatibility, etc.
//
type PitState struct {
	Data []byte `json:"data"`
}

func (vm *Vm) CreatePit() error {
	// Prepare the PIT config.
	// The only flag supported at the time of writing
	// was KVM_PIT_SPEAKER_DUMMY, which I really have no
	// interest in supporting.
	var pit C.struct_kvm_pit_config
	pit.flags = C.__u32(0)

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlCreatePit2),
		uintptr(unsafe.Pointer(&pit)))
	if e != 0 {
		return e
	}

	return nil
}

func (vm *Vm) GetPit() (PitState, error) {

	// Prepare the pit state.
	state := PitState{make([]byte, C.PitSize, C.PitSize)}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlGetPit2),
		uintptr(unsafe.Pointer(&state.Data[0])))
	if e != 0 {
		return state, e
	}

	return state, nil
}

func (vm *Vm) SetPit(state PitState) error {

	// Is there any state to set?
	// We just eat this error, it's fine.
	if state.Data == nil {
		return nil
	}

	// Is this the right size?
	if len(state.Data) != int(C.PitSize) {
		return PitIncompatible
	}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlSetPit2),
		uintptr(unsafe.Pointer(&state.Data[0])))
	if e != 0 {
		return e
	}

	return nil
}
