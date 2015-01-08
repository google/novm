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

import (
	"novmm/platform"
)

type Bios struct {
	BaseDevice

	// Our reserved TSS (for Intel VTX).
	TSSAddr platform.Paddr `json:"tss"`
}

func NewBios(info *DeviceInfo) (Device, error) {
	bios := new(Bios)

	// Sensible default.
	bios.TSSAddr = 0xfffbc000

	return bios, bios.init(info)
}

func (bios *Bios) Attach(vm *platform.Vm, model *Model) error {

	// Reserve our basic "BIOS" memory.
	// This is done simply to match expectations.
	// Nothing should be allocated in the first page.
	err := model.Reserve(
		vm,
		bios,
		MemoryTypeReserved,
		platform.Paddr(0), // Start.
		platform.PageSize, // Size.
		nil)
	if err != nil {
		return err
	}

	// Now reserve our TSS.
	err = model.Reserve(
		vm,
		bios,
		MemoryTypeSpecial,
		bios.TSSAddr,
		vm.SizeSpecialMemory(),
		nil)
	if err != nil {
		return err
	}

	// Finish the region.
	tss_end := bios.TSSAddr.After(vm.SizeSpecialMemory())
	err = model.Reserve(
		vm,
		bios,
		MemoryTypeReserved,
		tss_end,
		uint64(platform.Paddr(0x100000000)-tss_end),
		nil)
	if err != nil {
		return err
	}

	// We're good.
	return nil
}
