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

const int IoctlGetFpu = KVM_GET_FPU;
const int IoctlSetFpu = KVM_SET_FPU;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

//
// Our FPU state.
//
type Fpu struct {
	FPR  [8][16]uint8
	FCW  uint16
	FSW  uint16
	FTWX uint8

	LastOpcode uint16 `json:"last-opcode"`
	LastIp     uint64 `json:"last-ip"`
	LastDp     uint64 `json:"last-dp"`

	XMM   [16][16]uint8
	MXCSR uint32
}

func (vcpu *Vcpu) GetFpuState() (Fpu, error) {

	// Execute the ioctl.
	var kvm_fpu C.struct_kvm_fpu
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetFpu),
		uintptr(unsafe.Pointer(&kvm_fpu)))
	if e != 0 {
		return Fpu{}, e
	}

	// Transform our state.
	state := Fpu{}

	for i := 0; i < len(state.FPR); i += 1 {
		for j := 0; j < len(state.FPR[i]); j += 1 {
			state.FPR[i][j] = uint8(kvm_fpu.fpr[i][j])
		}
	}
	state.FCW = uint16(kvm_fpu.fcw)
	state.FSW = uint16(kvm_fpu.fsw)
	state.FTWX = uint8(kvm_fpu.ftwx)
	state.LastOpcode = uint16(kvm_fpu.last_opcode)
	state.LastIp = uint64(kvm_fpu.last_ip)
	state.LastDp = uint64(kvm_fpu.last_dp)
	for i := 0; i < len(state.XMM); i += 1 {
		for j := 0; j < len(state.XMM[i]); j += 1 {
			state.XMM[i][j] = uint8(kvm_fpu.xmm[i][j])
		}
	}
	state.MXCSR = uint32(kvm_fpu.mxcsr)

	return state, nil
}

func (vcpu *Vcpu) SetFpuState(state Fpu) error {

	// Prepare our data.
	var kvm_fpu C.struct_kvm_fpu
	for i := 0; i < len(state.FPR); i += 1 {
		for j := 0; j < len(state.FPR[i]); j += 1 {
			kvm_fpu.fpr[i][j] = C.__u8(state.FPR[i][j])
		}
	}
	kvm_fpu.fcw = C.__u16(state.FCW)
	kvm_fpu.fsw = C.__u16(state.FSW)
	kvm_fpu.ftwx = C.__u8(state.FTWX)
	kvm_fpu.last_opcode = C.__u16(state.LastOpcode)
	kvm_fpu.last_ip = C.__u64(state.LastIp)
	kvm_fpu.last_dp = C.__u64(state.LastDp)
	for i := 0; i < len(state.XMM); i += 1 {
		for j := 0; j < len(state.XMM[i]); j += 1 {
			kvm_fpu.xmm[i][j] = C.__u8(state.XMM[i][j])
		}
	}
	kvm_fpu.mxcsr = C.__u32(state.MXCSR)

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetFpu),
		uintptr(unsafe.Pointer(&kvm_fpu)))
	if e != 0 {
		return e
	}

	return nil
}
