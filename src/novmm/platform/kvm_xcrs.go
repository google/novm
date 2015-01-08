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
const int IoctlGetXcrs = KVM_GET_XCRS;
const int IoctlSetXcrs = KVM_SET_XCRS;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// A single XCR.
//
type Xcr struct {
	Id    uint32 `json:"xcr"`
	Value uint64 `json:"value"`
}

func (vcpu *Vcpu) GetXcrs() ([]Xcr, error) {

	// Execute the ioctl.
	var kvm_xcrs C.struct_kvm_xcrs
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetXcrs),
		uintptr(unsafe.Pointer(&kvm_xcrs)))
	if e != 0 {
		return nil, e
	}

	// Build our list.
	xcrs := make([]Xcr, 0, kvm_xcrs.nr_xcrs)
	for i := 0; i < int(kvm_xcrs.nr_xcrs); i += 1 {
		xcrs = append(xcrs, Xcr{
			Id:    uint32(kvm_xcrs.xcrs[i].xcr),
			Value: uint64(kvm_xcrs.xcrs[i].value),
		})
	}

	return xcrs, nil
}

func (vcpu *Vcpu) SetXcrs(xcrs []Xcr) error {

	// Build our parameter.
	var kvm_xcrs C.struct_kvm_xcrs
	kvm_xcrs.nr_xcrs = C.__u32(len(xcrs))
	for i, xcr := range xcrs {
		kvm_xcrs.xcrs[i].xcr = C.__u32(xcr.Id)
		kvm_xcrs.xcrs[i].value = C.__u64(xcr.Value)
	}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetXcrs),
		uintptr(unsafe.Pointer(&kvm_xcrs)))
	if e != 0 {
		return e
	}

	return nil
}
