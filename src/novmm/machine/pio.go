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

type PioEvent struct {
	*platform.ExitPio
}

func (pio PioEvent) Size() uint {
	return pio.ExitPio.Size()
}

func (pio PioEvent) GetData() uint64 {
	return *pio.ExitPio.Data()
}

func (pio PioEvent) SetData(val uint64) {
	*pio.ExitPio.Data() = val
}

func (pio PioEvent) IsWrite() bool {
	return pio.ExitPio.IsOut()
}

type PioDevice struct {
	BaseDevice

	// A map of our available ports.
	IoMap      `json:"-"`
	IoHandlers `json:"-"`

	// Our address in memory.
	Offset platform.Paddr `json:"base"`
}

func (pio *PioDevice) PioHandlers() IoHandlers {
	return pio.IoHandlers
}

func (pio *PioDevice) Attach(vm *platform.Vm, model *Model) error {

	// Build our IO Handlers.
	pio.IoHandlers = make(IoHandlers)
	for region, ops := range pio.IoMap {
		new_region := MemoryRegion{region.Start + pio.Offset, region.Size}
		pio.IoHandlers[new_region] = NewIoHandler(pio, new_region.Start, ops)
	}

	// NOTE: Unlike pio devices, we don't reserve
	// memory regions for our ports. Whichever device
	// gets there first wins.

	return pio.BaseDevice.Attach(vm, model)
}
