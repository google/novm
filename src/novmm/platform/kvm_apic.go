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
const int IoctlCreateIrqChip = KVM_CREATE_IRQCHIP;
const int IoctlGetIrqChip = KVM_GET_IRQCHIP;
const int IoctlSetIrqChip = KVM_SET_IRQCHIP;
const int IoctlIrqLine = KVM_IRQ_LINE;
const int IoctlSignalMsi = KVM_SIGNAL_MSI;
const int IoctlGetLApic = KVM_GET_LAPIC;
const int IoctlSetLApic = KVM_SET_LAPIC;

// Size of our lapic state.
const int ApicSize = KVM_APIC_REG_SIZE;

// We need to fudge the types for irq level.
// This is because of the extremely annoying semantics
// for accessing *unions* in Go. Basically it can't.
// See the description below in createIrqChip().
struct irq_level {
    __u32 irq;
    __u32 level;
};
static int check_irq_level(void) {
    if (sizeof(struct kvm_irq_level) != sizeof(struct irq_level)) {
        return 1;
    } else {
        return 0;
    }
}
*/
import "C"

import (
	"errors"
	"syscall"
	"unsafe"
)

//
// IrqChip --
//
// The IrqChip state requires three different
// devices: pic1, pic2 and the I/O apic. Each
// of these devices can be represented with a
// simple blob of data (compatibility will be
// the responsibility of KVM internally).
//
type IrqChip struct {
	Pic1   []byte `json:"pic1"`
	Pic2   []byte `json:"pic2"`
	IOApic []byte `json:"ioapic"`
}

//
// LApicState --
//
// Just a blob of data. KVM will be ensure
// forward-compatibility.
//
type LApicState struct {
	Data []byte `json:"data"`
}

func (vm *Vm) CreateIrqChip() error {
	// No parameters needed, just create the chip.
	// This is called as the VM is being created in
	// order to ensure that all future vcpus will have
	// their own local apic.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlCreateIrqChip),
		0)
	if e != 0 {
		return e
	}

	// Ugh. A bit of type-fudging. Because of the
	// way go handles unions, we use a custom type
	// for the Interrupt() function below. Let's just
	// check once that everything is sane.
	if C.check_irq_level() != 0 {
		return errors.New("KVM irq_level doesn't match expected!")
	}

	return nil
}

func LApic() Paddr {
	return Paddr(0xfee00000)
}

func IOApic() Paddr {
	return Paddr(0xfec00000)
}

func (vm *Vm) Interrupt(
	irq Irq,
	level bool) error {

	// Prepare the IRQ.
	var irq_level C.struct_irq_level
	irq_level.irq = C.__u32(irq)
	if level {
		irq_level.level = C.__u32(1)
	} else {
		irq_level.level = C.__u32(0)
	}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlIrqLine),
		uintptr(unsafe.Pointer(&irq_level)))
	if e != 0 {
		return e
	}

	return nil
}

func (vm *Vm) SignalMSI(
	addr Paddr,
	data uint32,
	flags uint32) error {

	// Prepare the MSI.
	var msi C.struct_kvm_msi
	msi.address_lo = C.__u32(addr & 0xffffffff)
	msi.address_hi = C.__u32(addr >> 32)
	msi.data = C.__u32(data)
	msi.flags = C.__u32(flags)

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlSignalMsi),
		uintptr(unsafe.Pointer(&msi)))
	if e != 0 {
		return e
	}

	return nil
}

func (vm *Vm) GetIrqChip() (IrqChip, error) {

	var state IrqChip

	// Create our scratch buffer.
	// The expected layout of the structure is:
	//  u32     - chip_id
	//  u32     - pad
	//  byte[0] - data
	buf := make([]byte, PageSize, PageSize)

	for i := 0; i < 3; i += 1 {

		// Set our chip_id.
		buf[0] = byte(i)

		// Execute the ioctl.
		_, _, e := syscall.Syscall(
			syscall.SYS_IOCTL,
			uintptr(vm.fd),
			uintptr(C.IoctlGetIrqChip),
			uintptr(unsafe.Pointer(&buf[0])))
		if e != 0 {
			return state, e
		}

		// Copy appropriate state out.
		switch buf[0] {
		case 0:
			state.Pic1 = make([]byte, len(buf)-8, len(buf)-8)
			copy(state.Pic1, buf[8:])
		case 1:
			state.Pic2 = make([]byte, len(buf)-8, len(buf)-8)
			copy(state.Pic2, buf[8:])
		case 2:
			state.IOApic = make([]byte, len(buf)-8, len(buf)-8)
			copy(state.IOApic, buf[8:])
		}
	}

	return state, nil
}

func (vm *Vm) SetIrqChip(state IrqChip) error {

	// Create our scratch buffer.
	// See GetIrqChip for expected layout.
	buf := make([]byte, PageSize, PageSize)

	for i := 0; i < 3; i += 1 {

		// Set our chip_id.
		buf[0] = byte(i)

		// Copy appropriate state in.
		// We also ensure that we have the
		// appropriate state to load. If we don't
		// it's fine, we just continue along.
		switch i {
		case 0:
			if state.Pic1 == nil {
				continue
			}
			copy(buf[8:], state.Pic1)
		case 1:
			if state.Pic2 == nil {
				continue
			}
			copy(buf[8:], state.Pic2)
		case 2:
			if state.IOApic == nil {
				continue
			}
			copy(buf[8:], state.IOApic)
		}

		// Execute the ioctl.
		_, _, e := syscall.Syscall(
			syscall.SYS_IOCTL,
			uintptr(vm.fd),
			uintptr(C.IoctlSetIrqChip),
			uintptr(unsafe.Pointer(&buf[0])))
		if e != 0 {
			return e
		}
	}

	return nil
}

func (vcpu *Vcpu) GetLApic() (LApicState, error) {
	// Prepare the apic state.
	state := LApicState{make([]byte, C.ApicSize, C.ApicSize)}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlGetLApic),
		uintptr(unsafe.Pointer(&state.Data[0])))
	if e != 0 {
		return state, e
	}

	return state, nil
}

func (vcpu *Vcpu) SetLApic(state LApicState) error {

	// Is there any state to set?
	// We just eat this error, it's fine.
	if state.Data == nil {
		return nil
	}

	// Check the state is reasonable.
	if len(state.Data) != int(C.ApicSize) {
		return LApicIncompatible
	}

	// Execute the ioctl.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vcpu.fd),
		uintptr(C.IoctlSetLApic),
		uintptr(unsafe.Pointer(&state.Data[0])))
	if e != 0 {
		return e
	}

	return nil
}
