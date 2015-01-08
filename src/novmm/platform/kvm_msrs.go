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
#include "kvm_msrs.h"

const int IoctlGetMsrIndexList = KVM_GET_MSR_INDEX_LIST;
const int IoctlSetMsrs = KVM_SET_MSRS;
const int IoctlGetMsrs = KVM_GET_MSRS;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

type Msr struct {
	Index uint32 `json:"index"`
	Value uint64 `json:"value"`
}

func availableMsrs(fd int) ([]uint32, error) {

	// Find our list of MSR indices.
	// A page should be more than enough here,
	// eventually if it's not we'll end up with
	// a failed system call for some reason other
	// than E2BIG (which just says n is wrong).
	msrIndices := make([]byte, PageSize, PageSize)
	msrs := make([]uint32, 0, 0)

	for {
		_, _, e := syscall.Syscall(
			syscall.SYS_IOCTL,
			uintptr(fd),
			uintptr(C.IoctlGetMsrIndexList),
			uintptr(unsafe.Pointer(&msrIndices[0])))
		if e == syscall.E2BIG {
			// The nmsrs field will now have been
			// adjusted, and we can run it again.
			continue
		} else if e != 0 {
			return nil, e
		}

		// We're good!
		break
	}

	// Extract each msr individually.
	for i := 0; ; i += 1 {
		// Is there a valid index?
		var index C.__u32
		e := C.msr_list_index(
			unsafe.Pointer(&msrIndices[0]),
			C.int(i),
			&index)

		// Any left?
		if e != 0 {
			break
		}

		// Add this MSR.
		msrs = append(msrs, uint32(index))
	}

	return msrs, nil
}

func (vcpu *Vcpu) GetMsr(index uint32) (uint64, error) {

	// Setup our structure.
	data := make([]byte, C.msr_size(), C.msr_size())

	// Set our index to retrieve.
	C.msr_set(unsafe.Pointer(&data[0]), C.__u32(index), C.__u64(0))

	// Execute our ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetMsrs),
		uintptr(unsafe.Pointer(&data[0])))
	if e != 0 {
		return 0, e
	}

	// Return our value.
	return uint64(C.msr_get(unsafe.Pointer(&data[0]))), nil
}

func (vcpu *Vcpu) SetMsr(index uint32, value uint64) error {

	// Setup our structure.
	data := make([]byte, C.msr_size(), C.msr_size())

	// Set our index and value.
	C.msr_set(unsafe.Pointer(&data[0]), C.__u32(index), C.__u64(value))

	// Execute our ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetMsrs),
		uintptr(unsafe.Pointer(&data[0])))
	if e != 0 {
		return e
	}

	return nil
}

func (vcpu *Vcpu) GetMsrs() ([]Msr, error) {

	// Extract each msr individually.
	msrs := make([]Msr, 0, len(vcpu.msrs))

	for _, index := range vcpu.msrs {

		// Get this MSR.
		value, err := vcpu.GetMsr(index)
		if err != nil {
			return msrs, err
		}

		// Got one.
		msrs = append(msrs, Msr{uint32(index), uint64(value)})
	}

	// Finish it off.
	return msrs, nil
}

func (vcpu *Vcpu) SetMsrs(msrs []Msr) error {

	for _, msr := range msrs {
		// Set our msrs.
		err := vcpu.SetMsr(msr.Index, msr.Value)
		if err != nil {
			return err
		}
	}

	return nil
}
