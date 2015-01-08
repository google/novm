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
const int IoctlGetVcpuEvents = KVM_GET_VCPU_EVENTS;
const int IoctlSetVcpuEvents = KVM_SET_VCPU_EVENTS;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// Our event state.
//
type ExceptionEvent struct {
	Number    uint8   `json:"number"`
	ErrorCode *uint32 `json:"error-code"`
}

type InterruptEvent struct {
	Number uint8 `json:"number"`
	Soft   bool  `json:"soft"`
	Shadow bool  `json:"shadow"`
}

type Events struct {
	Exception *ExceptionEvent `json:"exception"`
	Interrupt *InterruptEvent `json:"interrupt"`

	NmiPending bool `json:"nmi-pending"`
	NmiMasked  bool `json:"nmi-masked"`

	SipiVector uint32 `json:"sipi-vector"`
	Flags      uint32 `json:"flags"`
}

func (vcpu *Vcpu) GetEvents() (Events, error) {

	// Execute the ioctl.
	var kvm_events C.struct_kvm_vcpu_events
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetVcpuEvents),
		uintptr(unsafe.Pointer(&kvm_events)))
	if e != 0 {
		return Events{}, e
	}

	// Prepare our state.
	events := Events{
		NmiPending: kvm_events.nmi.pending != C.__u8(0),
		NmiMasked:  kvm_events.nmi.masked != C.__u8(0),
		SipiVector: uint32(kvm_events.sipi_vector),
		Flags:      uint32(kvm_events.flags),
	}
	if kvm_events.exception.injected != C.__u8(0) {
		events.Exception = &ExceptionEvent{
			Number: uint8(kvm_events.exception.nr),
		}
		if kvm_events.exception.has_error_code != C.__u8(0) {
			error_code := uint32(kvm_events.exception.error_code)
			events.Exception.ErrorCode = &error_code
		}
	}
	if kvm_events.interrupt.injected != C.__u8(0) {
		events.Interrupt = &InterruptEvent{
			Number: uint8(kvm_events.interrupt.nr),
			Soft:   kvm_events.interrupt.soft != C.__u8(0),
			Shadow: kvm_events.interrupt.shadow != C.__u8(0),
		}
	}

	return events, nil
}

func (vcpu *Vcpu) SetEvents(events Events) error {

	// Prepare our state.
	var kvm_events C.struct_kvm_vcpu_events

	if events.NmiPending {
		kvm_events.nmi.pending = C.__u8(1)
	} else {
		kvm_events.nmi.pending = C.__u8(0)
	}
	if events.NmiMasked {
		kvm_events.nmi.masked = C.__u8(1)
	} else {
		kvm_events.nmi.masked = C.__u8(0)
	}

	kvm_events.sipi_vector = C.__u32(events.SipiVector)
	kvm_events.flags = C.__u32(events.Flags)

	if events.Exception != nil {
		kvm_events.exception.injected = C.__u8(1)
		kvm_events.exception.nr = C.__u8(events.Exception.Number)
		if events.Exception.ErrorCode != nil {
			kvm_events.exception.has_error_code = C.__u8(1)
			kvm_events.exception.error_code = C.__u32(*events.Exception.ErrorCode)
		} else {
			kvm_events.exception.has_error_code = C.__u8(0)
			kvm_events.exception.error_code = C.__u32(0)
		}
	} else {
		kvm_events.exception.injected = C.__u8(0)
		kvm_events.exception.nr = C.__u8(0)
		kvm_events.exception.has_error_code = C.__u8(0)
		kvm_events.exception.error_code = C.__u32(0)
	}
	if events.Interrupt != nil {
		kvm_events.interrupt.injected = C.__u8(1)
		kvm_events.interrupt.nr = C.__u8(events.Interrupt.Number)
		if events.Interrupt.Soft {
			kvm_events.interrupt.soft = C.__u8(1)
		} else {
			kvm_events.interrupt.soft = C.__u8(0)
		}
		if events.Interrupt.Shadow {
			kvm_events.interrupt.shadow = C.__u8(1)
		} else {
			kvm_events.interrupt.shadow = C.__u8(0)
		}
	} else {
		kvm_events.interrupt.injected = C.__u8(0)
		kvm_events.interrupt.nr = C.__u8(0)
		kvm_events.interrupt.soft = C.__u8(0)
		kvm_events.interrupt.shadow = C.__u8(0)
	}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetVcpuEvents),
		uintptr(unsafe.Pointer(&kvm_events)))
	if e != 0 {
		return e
	}

	return nil
}
