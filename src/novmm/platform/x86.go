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

// +build i386 amd64
package platform

//
// x86 platform constants.
//
const (
	PageSize = 4096
)

//
// Our general purpose registers.
//
type Register int
type RegisterValue uint64

const (
	RAX Register = iota
	RBX
	RCX
	RDX
	RSI
	RDI
	RSP
	RBP
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15
	RIP
	RFLAGS
)

//
// Special control registers.
//
type ControlRegister int
type ControlRegisterValue uint64

const (
	CR0 ControlRegister = iota
	CR2
	CR3
	CR4
	CR8
	EFER
	APIC_BASE
)

//
// Segment descriptor registers.
//
type Descriptor int
type DescriptorValue struct {
	Base  uint64 `json:"base"`
	Limit uint16 `json:"limit"`
}

const (
	GDT Descriptor = iota
	IDT
)

//
// Segment registers.
//
type Segment int
type SegmentValue struct {
	Base     uint64 `json:"base"`
	Limit    uint32 `json:"limit"`
	Selector uint16 `json:"selector"`
	Type     uint8  `json:"type"`
	Present  uint8  `json:"present"`
	Dpl      uint8  `json:"dpl"`
	Db       uint8  `json:"db"`
	L        uint8  `json:"l"`
	S        uint8  `json:"s"`
	G        uint8  `json:"g"`
	Avl      uint8  `json:"avl"`
}

const (
	CS Segment = iota
	DS
	ES
	FS
	GS
	SS
	TR
	LDT
)

//
// Utility structure containing all registers.
//
type Registers struct {
	RAX    *RegisterValue
	RBX    *RegisterValue
	RCX    *RegisterValue
	RDX    *RegisterValue
	RSI    *RegisterValue
	RDI    *RegisterValue
	RSP    *RegisterValue
	RBP    *RegisterValue
	R8     *RegisterValue
	R9     *RegisterValue
	R10    *RegisterValue
	R11    *RegisterValue
	R12    *RegisterValue
	R13    *RegisterValue
	R14    *RegisterValue
	R15    *RegisterValue
	RIP    *RegisterValue
	RFLAGS *RegisterValue

	CR0       *ControlRegisterValue
	CR2       *ControlRegisterValue
	CR3       *ControlRegisterValue
	CR4       *ControlRegisterValue
	CR8       *ControlRegisterValue
	EFER      *ControlRegisterValue
	APIC_BASE *ControlRegisterValue `json:"APIC"`

	IDT *DescriptorValue
	GDT *DescriptorValue

	CS  *SegmentValue
	DS  *SegmentValue
	ES  *SegmentValue
	FS  *SegmentValue
	GS  *SegmentValue
	SS  *SegmentValue
	TR  *SegmentValue
	LDT *SegmentValue
}

func (vcpu *Vcpu) getRegister(
	name string,
	reg Register,
	errs []error) (*RegisterValue, []error) {

	value, err := vcpu.GetRegister(reg)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	return &value, errs
}

func (vcpu *Vcpu) getControlRegister(
	name string,
	reg ControlRegister,
	errs []error) (*ControlRegisterValue, []error) {

	value, err := vcpu.GetControlRegister(reg)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	return &value, errs
}

func (vcpu *Vcpu) getDescriptor(
	name string,
	desc Descriptor,
	errs []error) (*DescriptorValue, []error) {

	value, err := vcpu.GetDescriptor(desc)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	return &value, errs
}

func (vcpu *Vcpu) getSegment(
	name string,
	seg Segment,
	errs []error) (*SegmentValue, []error) {

	value, err := vcpu.GetSegment(seg)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	return &value, errs
}

func (vcpu *Vcpu) GetRegisters() (Registers, error) {
	vcpu.Pause(false)
	defer vcpu.Unpause(false)

	var regs Registers
	errs := make([]error, 0, 0)

	regs.RAX, errs = vcpu.getRegister("RAX", RAX, errs)
	regs.RBX, errs = vcpu.getRegister("RBX", RBX, errs)
	regs.RCX, errs = vcpu.getRegister("RCX", RCX, errs)
	regs.RDX, errs = vcpu.getRegister("RDX", RDX, errs)
	regs.RSI, errs = vcpu.getRegister("RSI", RSI, errs)
	regs.RDI, errs = vcpu.getRegister("RDI", RDI, errs)
	regs.RSP, errs = vcpu.getRegister("RSP", RSP, errs)
	regs.RBP, errs = vcpu.getRegister("RBP", RBP, errs)
	regs.R8, errs = vcpu.getRegister("R8", R8, errs)
	regs.R9, errs = vcpu.getRegister("R9", R9, errs)
	regs.R10, errs = vcpu.getRegister("R10", R10, errs)
	regs.R11, errs = vcpu.getRegister("R11", R11, errs)
	regs.R12, errs = vcpu.getRegister("R12", R12, errs)
	regs.R13, errs = vcpu.getRegister("R13", R13, errs)
	regs.R14, errs = vcpu.getRegister("R14", R14, errs)
	regs.R15, errs = vcpu.getRegister("R15", R15, errs)
	regs.RIP, errs = vcpu.getRegister("RIP", RIP, errs)
	regs.RFLAGS, errs = vcpu.getRegister("RFLAGS", RFLAGS, errs)

	regs.CR0, errs = vcpu.getControlRegister("CR0", CR0, errs)
	regs.CR2, errs = vcpu.getControlRegister("CR2", CR2, errs)
	regs.CR3, errs = vcpu.getControlRegister("CR3", CR3, errs)
	regs.CR4, errs = vcpu.getControlRegister("CR4", CR4, errs)
	regs.CR8, errs = vcpu.getControlRegister("CR8", CR8, errs)
	regs.EFER, errs = vcpu.getControlRegister("EFER", EFER, errs)
	regs.APIC_BASE, errs = vcpu.getControlRegister("APIC_BASE", APIC_BASE, errs)

	regs.GDT, errs = vcpu.getDescriptor("GDT", GDT, errs)
	regs.IDT, errs = vcpu.getDescriptor("IDT", IDT, errs)

	regs.CS, errs = vcpu.getSegment("CS", CS, errs)
	regs.DS, errs = vcpu.getSegment("DS", DS, errs)
	regs.ES, errs = vcpu.getSegment("ES", ES, errs)
	regs.FS, errs = vcpu.getSegment("FS", FS, errs)
	regs.GS, errs = vcpu.getSegment("GS", GS, errs)
	regs.SS, errs = vcpu.getSegment("SS", SS, errs)
	regs.TR, errs = vcpu.getSegment("TR", TR, errs)
	regs.LDT, errs = vcpu.getSegment("LDT", LDT, errs)

	// Return a simple error.
	// We could actually return a more
	// meaningful error here that describes
	// all the registers which had errors,
	// but for now this will do the trick.
	for _, err := range errs {
		if err != nil {
			return Registers{}, err
		}
	}

	return regs, nil
}

func (vcpu *Vcpu) setRegister(
	name string,
	reg Register,
	value *RegisterValue,
	errs []error) []error {

	if value != nil {
		err := vcpu.SetRegister(reg, *value)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (vcpu *Vcpu) setControlRegister(
	name string,
	reg ControlRegister,
	value *ControlRegisterValue,
	errs []error) []error {

	if value != nil {
		err := vcpu.SetControlRegister(reg, *value, false)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (vcpu *Vcpu) setDescriptor(
	name string,
	desc Descriptor,
	value *DescriptorValue,
	errs []error) []error {

	if value != nil {
		err := vcpu.SetDescriptor(desc, *value, false)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (vcpu *Vcpu) setSegment(
	name string,
	seg Segment,
	value *SegmentValue,
	errs []error) []error {

	if value != nil {
		err := vcpu.SetSegment(seg, *value, false)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (vcpu *Vcpu) SetRegisters(regs Registers) error {
	vcpu.Pause(false)
	defer vcpu.Unpause(false)

	errs := make([]error, 0, 0)

	errs = vcpu.setSegment("CS", CS, regs.CS, errs)
	errs = vcpu.setSegment("DS", DS, regs.DS, errs)
	errs = vcpu.setSegment("ES", ES, regs.ES, errs)
	errs = vcpu.setSegment("FS", FS, regs.FS, errs)
	errs = vcpu.setSegment("GS", GS, regs.GS, errs)
	errs = vcpu.setSegment("SS", SS, regs.SS, errs)
	errs = vcpu.setSegment("TR", TR, regs.TR, errs)
	errs = vcpu.setSegment("LDT", LDT, regs.LDT, errs)

	errs = vcpu.setRegister("RAX", RAX, regs.RAX, errs)
	errs = vcpu.setRegister("RBX", RBX, regs.RBX, errs)
	errs = vcpu.setRegister("RCX", RCX, regs.RCX, errs)
	errs = vcpu.setRegister("RDX", RDX, regs.RDX, errs)
	errs = vcpu.setRegister("RSI", RSI, regs.RSI, errs)
	errs = vcpu.setRegister("RDI", RDI, regs.RDI, errs)
	errs = vcpu.setRegister("RSP", RSP, regs.RSP, errs)
	errs = vcpu.setRegister("RBP", RBP, regs.RBP, errs)
	errs = vcpu.setRegister("R8", R8, regs.R8, errs)
	errs = vcpu.setRegister("R9", R9, regs.R9, errs)
	errs = vcpu.setRegister("R10", R10, regs.R10, errs)
	errs = vcpu.setRegister("R11", R11, regs.R11, errs)
	errs = vcpu.setRegister("R12", R12, regs.R12, errs)
	errs = vcpu.setRegister("R13", R13, regs.R13, errs)
	errs = vcpu.setRegister("R14", R14, regs.R14, errs)
	errs = vcpu.setRegister("R15", R15, regs.R15, errs)
	errs = vcpu.setRegister("RIP", RIP, regs.RIP, errs)
	errs = vcpu.setRegister("RFLAGS", RFLAGS, regs.RFLAGS, errs)

	errs = vcpu.setControlRegister("CR0", CR0, regs.CR0, errs)
	errs = vcpu.setControlRegister("CR2", CR2, regs.CR2, errs)
	errs = vcpu.setControlRegister("CR3", CR3, regs.CR3, errs)
	errs = vcpu.setControlRegister("CR4", CR4, regs.CR4, errs)
	errs = vcpu.setControlRegister("CR8", CR8, regs.CR8, errs)
	errs = vcpu.setControlRegister("EFER", EFER, regs.EFER, errs)
	errs = vcpu.setControlRegister("APIC_BASE", APIC_BASE, regs.APIC_BASE, errs)

	errs = vcpu.setDescriptor("GDT", GDT, regs.GDT, errs)
	errs = vcpu.setDescriptor("IDT", IDT, regs.IDT, errs)

	// As per GetRegisters(), return a simple error.
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return vcpu.flushAllRegs()
}
