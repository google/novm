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

import (
	"log"
)

type VcpuInfo struct {

	// Our optional id.
	// If this is not provided, we
	// assume that it is in order.
	Id *uint `json:"id"`

	// Full register state.
	Registers Registers `json:"registers"`

	// Optional multiprocessor state.
	MpState *MpState `json:"state"`

	// Our cpuid (not optional).
	Cpuid []Cpuid `json:"cpuid"`

	// Our LApic state.
	// This is optional, but is handled
	// within kvm_apic.go and not here.
	LApic LApicState `json:"lapic"`

	// Our msrs (not optional).
	Msrs []Msr `json:"msrs"`

	// Our pending vcpu events.
	Events Events `json:"events"`

	// Optional FRU state.
	Fpu *Fpu `json:"fpu"`

	// Extended control registers.
	Xcrs []Xcr `json:"xcrs"`

	// Optional xsave state.
	XSave *XSave `json:"xsave"`
}

func (vm *Vm) CreateVcpus(spec []VcpuInfo) ([]*Vcpu, error) {

	vcpus := make([]*Vcpu, 0, 0)

	// Load all vcpus.
	for index, info := range spec {

		// Sanitize vcpu ids.
		if info.Id == nil {
			newid := uint(index)
			info.Id = &newid
		}

		// Create a new vcpu.
		vcpu, err := vm.NewVcpu(*info.Id)
		if err != nil {
			return nil, err
		}

		// Load the state.
		err = vcpu.Load(info)
		if err != nil {
			return nil, err
		}

		// Good to go.
		vcpus = append(vcpus, vcpu)
	}

	// We've okay.
	return vcpus, nil
}

func (vcpu *Vcpu) Load(info VcpuInfo) error {

	// Ensure the registers are loaded.
	log.Printf("vcpu[%d]: setting registers...", vcpu.Id)
	vcpu.SetRegisters(info.Registers)

	// Optional multiprocessing state.
	if info.MpState != nil {
		log.Printf("vcpu[%d]: setting vcpu state...", vcpu.Id)
		err := vcpu.SetMpState(*info.MpState)
		if err != nil {
			return err
		}
	}

	// Set our cpuid if we have one.
	if info.Cpuid != nil {
		log.Printf("vcpu[%d]: setting cpuid...", vcpu.Id)
		err := vcpu.SetCpuid(info.Cpuid)
		if err != nil {
			return err
		}
	}

	// Always load our Lapic.
	log.Printf("vcpu[%d]: setting apic state...", vcpu.Id)
	err := vcpu.SetLApic(info.LApic)
	if err != nil {
		return err
	}

	// Load MSRs if available.
	if info.Msrs != nil {
		log.Printf("vcpu[%d]: setting msrs...", vcpu.Id)
		err := vcpu.SetMsrs(info.Msrs)
		if err != nil {
			return err
		}
	}

	// Load events.
	log.Printf("vcpu[%d]: setting vcpu events...", vcpu.Id)
	err = vcpu.SetEvents(info.Events)
	if err != nil {
		return err
	}

	// Load fpu state if available.
	if info.Fpu != nil {
		log.Printf("vcpu[%d]: setting fpu state...", vcpu.Id)
		err = vcpu.SetFpuState(*info.Fpu)
		if err != nil {
			return err
		}
	}

	// Load Xcrs if available.
	if info.Xcrs != nil {
		log.Printf("vcpu[%d]: setting xcrs...", vcpu.Id)
		err = vcpu.SetXcrs(info.Xcrs)
		if err != nil {
			return err
		}
	}

	// Load xsave state if available.
	if info.XSave != nil {
		log.Printf("vcpu[%d]: setting xsave state...", vcpu.Id)
		err = vcpu.SetXSave(*info.XSave)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewVcpuInfo(vcpu *Vcpu) (VcpuInfo, error) {

	err := vcpu.Pause(false)
	if err != nil {
		return VcpuInfo{}, err
	}
	defer vcpu.Unpause(false)

	registers, err := vcpu.GetRegisters()
	if err != nil {
		return VcpuInfo{}, err
	}

	mpstate, err := vcpu.GetMpState()
	if err != nil {
		return VcpuInfo{}, err
	}

	cpuid, err := vcpu.GetCpuid()
	if err != nil {
		return VcpuInfo{}, err
	}

	lapic, err := vcpu.GetLApic()
	if err != nil {
		return VcpuInfo{}, err
	}

	msrs, err := vcpu.GetMsrs()
	if err != nil {
		return VcpuInfo{}, err
	}

	events, err := vcpu.GetEvents()
	if err != nil {
		return VcpuInfo{}, err
	}

	fpu, err := vcpu.GetFpuState()
	if err != nil {
		return VcpuInfo{}, err
	}

	xcrs, err := vcpu.GetXcrs()
	if err != nil {
		return VcpuInfo{}, err
	}

	xsave, err := vcpu.GetXSave()
	if err != nil {
		return VcpuInfo{}, err
	}

	return VcpuInfo{
		Id:        &vcpu.Id,
		Registers: registers,
		MpState:   &mpstate,
		Cpuid:     cpuid,
		LApic:     lapic,
		Msrs:      msrs,
		Events:    events,
		Fpu:       &fpu,
		Xcrs:      xcrs,
		XSave:     &xsave,
	}, nil
}
