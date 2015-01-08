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
const int IoctlGetXSave = KVM_GET_XSAVE;
const int IoctlSetXSave = KVM_SET_XSAVE;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// Our xsave state.
//
type XSave struct {
	Region [1024]uint32 `json:"region"`
}

func (vcpu *Vcpu) GetXSave() (XSave, error) {

	// Execute the ioctl.
	var kvm_xsave C.struct_kvm_xsave
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetXSave),
		uintptr(unsafe.Pointer(&kvm_xsave)))
	if e != 0 {
		return XSave{}, e
	}

	state := XSave{}
	for i := 0; i < len(state.Region); i += 1 {
		state.Region[i] = uint32(kvm_xsave.region[i])
	}

	return state, nil
}

func (vcpu *Vcpu) SetXSave(state XSave) error {

	// Execute the ioctl.
	var kvm_xsave C.struct_kvm_xsave
	for i := 0; i < len(state.Region); i += 1 {
		kvm_xsave.region[i] = C.__u32(state.Region[i])
	}
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetXSave),
		uintptr(unsafe.Pointer(&kvm_xsave)))
	if e != 0 {
		return e
	}

	return nil
}
