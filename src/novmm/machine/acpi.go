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

package machine

/*
#include "acpi.h"
*/
import "C"

import (
	"novmm/platform"
	"unsafe"
)

type Acpi struct {
	BaseDevice

	Addr platform.Paddr `json:"address"`
	Data []byte         `json:"data"`
}

func NewAcpi(info *DeviceInfo) (Device, error) {
	acpi := new(Acpi)
	acpi.Addr = platform.Paddr(0xf0000)
	return acpi, acpi.init(info)
}

func (acpi *Acpi) Attach(vm *platform.Vm, model *Model) error {

	// Do we already have data?
	rebuild := true
	if acpi.Data == nil {
		// Create our data.
		acpi.Data = make([]byte, platform.PageSize, platform.PageSize)
	} else {
		// Align our data.
		// This is necessary because we map this in
		// directly. It's possible that the data was
		// decoded and refers to the middle of some
		// larger array somewhere, and isn't aligned.
		acpi.Data = platform.AlignBytes(acpi.Data)
		rebuild = false
	}

	// Allocate our memory block.
	err := model.Reserve(
		vm,
		acpi,
		MemoryTypeAcpi,
		acpi.Addr,
		platform.PageSize,
		acpi.Data)
	if err != nil {
		return err
	}

	// Already done.
	if !rebuild {
		return nil
	}

	// Find our APIC information.
	// This will find the APIC device if it
	// is attached, otherwise the MADT table
	// will unfortunately have be a bit invalid.
	var IOApic platform.Paddr
	var LApic platform.Paddr
	for _, device := range model.Devices() {
		apic, ok := device.(*Apic)
		if ok {
			IOApic = apic.IOApic
			LApic = apic.LApic
			break
		}
	}

	// Load the MADT.
	madt_bytes := C.build_madt(
		unsafe.Pointer(&acpi.Data[0]),
		C.__u32(LApic),
		C.int(len(vm.Vcpus())),
		C.__u32(IOApic),
		C.__u32(0), // I/O APIC interrupt?
	)
	acpi.Debug("MADT %x @ %x", madt_bytes, acpi.Addr)

	// Align offset.
	offset := madt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the DSDT.
	dsdt_address := uint64(acpi.Addr) + uint64(offset)
	dsdt_bytes := C.build_dsdt(
		unsafe.Pointer(&acpi.Data[int(offset)]),
	)
	acpi.Debug("DSDT %x @ %x", dsdt_bytes, dsdt_address)

	// Align offset.
	offset += dsdt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the XSDT.
	xsdt_address := uint64(acpi.Addr) + uint64(offset)
	xsdt_bytes := C.build_xsdt(
		unsafe.Pointer(&acpi.Data[int(offset)]),
		C.__u64(acpi.Addr), // MADT address.
	)
	acpi.Debug("XSDT %x @ %x", xsdt_bytes, xsdt_address)

	// Align offset.
	offset += xsdt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the RSDT.
	rsdt_address := uint64(acpi.Addr) + uint64(offset)
	rsdt_bytes := C.build_rsdt(
		unsafe.Pointer(&acpi.Data[int(offset)]),
		C.__u32(acpi.Addr), // MADT address.
	)
	acpi.Debug("RSDT %x @ %x", rsdt_bytes, rsdt_address)

	// Align offset.
	offset += rsdt_bytes
	if offset%64 != 0 {
		offset += 64 - (offset % 64)
	}

	// Load the RSDP.
	rsdp_address := uint64(acpi.Addr) + uint64(offset)
	rsdp_bytes := C.build_rsdp(
		unsafe.Pointer(&acpi.Data[int(offset)]),
		C.__u32(rsdt_address), // RSDT address.
		C.__u64(xsdt_address), // XSDT address.
	)
	acpi.Debug("RSDP %x @ %x", rsdp_bytes, rsdp_address)

	// Everything went okay.
	return nil
}
