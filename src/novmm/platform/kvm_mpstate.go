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
#include "cpuid.h"

// IOCTL calls.
const int IoctlGetMpState = KVM_GET_MP_STATE;
const int IoctlSetMpState = KVM_SET_MP_STATE;

// States.
const int MpStateRunnable = KVM_MP_STATE_RUNNABLE;
const int MpStateUninitialized = KVM_MP_STATE_UNINITIALIZED;
const int MpStateInitReceived = KVM_MP_STATE_INIT_RECEIVED;
const int MpStateHalted = KVM_MP_STATE_HALTED;
const int MpStateSipiReceived = KVM_MP_STATE_SIPI_RECEIVED;
*/
import "C"

import (
	"encoding/json"
	"syscall"
	"unsafe"
)

//
// Our vcpus state.
//
type MpState C.int

var MpStateRunnable = MpState(C.MpStateRunnable)
var MpStateUninitialized = MpState(C.MpStateUninitialized)
var MpStateInitReceived = MpState(C.MpStateInitReceived)
var MpStateHalted = MpState(C.MpStateHalted)
var MpStateSipiReceived = MpState(C.MpStateSipiReceived)

var stateMap = map[MpState]string{
	MpStateRunnable:      "runnable",
	MpStateUninitialized: "uninitialized",
	MpStateInitReceived:  "init-received",
	MpStateHalted:        "halted",
	MpStateSipiReceived:  "sipi-received",
}

var stateRevMap = map[string]MpState{
	"runnable":      MpStateRunnable,
	"uninitialized": MpStateUninitialized,
	"init-received": MpStateInitReceived,
	"halted":        MpStateHalted,
	"sipi-received": MpStateSipiReceived,
}

func (vcpu *Vcpu) GetMpState() (MpState, error) {

	// Execute the ioctl.
	var kvm_state C.struct_kvm_mp_state
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetMpState),
		uintptr(unsafe.Pointer(&kvm_state)))
	if e != 0 {
		return MpState(kvm_state.mp_state), e
	}

	return MpState(kvm_state.mp_state), nil
}

func (vcpu *Vcpu) SetMpState(state MpState) error {

	// Execute the ioctl.
	var kvm_state C.struct_kvm_mp_state
	kvm_state.mp_state = C.__u32(state)
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetMpState),
		uintptr(unsafe.Pointer(&kvm_state)))
	if e != 0 {
		return e
	}

	return nil
}

func (state *MpState) MarshalJSON() ([]byte, error) {

	// Marshal as a string.
	value, ok := stateMap[*state]
	if !ok {
		return nil, UnknownState
	}

	return json.Marshal(value)
}

func (state *MpState) UnmarshalJSON(data []byte) error {

	// Unmarshal as an string.
	var value string
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	// Find the state.
	newstate, ok := stateRevMap[value]
	if !ok {
		return UnknownState
	}

	// That's our state.
	*state = newstate
	return nil
}
