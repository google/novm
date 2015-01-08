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

//
// Model --
//
// Our basic machine model.
//
// This is very much different from a standard virtual machine.
// First, we only support a very limited selection of devices.
// We actually do not support *any* I/O-port based devices, which
// includes PCI devices (which require an I/O port at the root).

type Model struct {

	// Basic memory layout:
	// This is generally accessible from the loader,
	// and other modules that may need to tweak memory.
	MemoryMap

	// Basic interrupt layout:
	// This maps interrupts to devices.
	InterruptMap

	// All devices.
	devices []Device

	// Our device lookup cache.
	pio_cache  *IoCache
	mmio_cache *IoCache
}

func NewModel(vm *platform.Vm) (*Model, error) {

	// Create our model object.
	model := new(Model)

	// Setup the memory map.
	model.MemoryMap = make(MemoryMap, 0, 0)

	// Setup the interrupt map.
	model.InterruptMap = make(InterruptMap)

	// Create our devices.
	model.devices = make([]Device, 0, 0)

	// We're set.
	return model, nil
}

func (model *Model) flush() error {

	collectIoHandlers := func(is_pio bool) []IoHandlers {
		io_handlers := make([]IoHandlers, 0, 0)
		for _, device := range model.devices {
			if is_pio {
				io_handlers = append(io_handlers, device.PioHandlers())
			} else {
				io_handlers = append(io_handlers, device.MmioHandlers())
			}
		}
		return io_handlers
	}

	// (Re-)Create our IoCache.
	model.pio_cache = NewIoCache(collectIoHandlers(true), true)
	model.mmio_cache = NewIoCache(collectIoHandlers(false), false)

	// We're okay.
	return nil
}

func (model *Model) Devices() []Device {
	return model.devices
}

func (model *Model) Pause(manual bool) error {

	for i, device := range model.devices {
		// Ensure all devices are paused.
		err := device.Pause(manual)
		if err != nil && err != DeviceAlreadyPaused {
			for i -= 1; i >= 0; i -= 1 {
				device.Unpause(manual)
			}
			return err
		}
	}

	// All good.
	return nil
}

func (model *Model) Unpause(manual bool) error {

	for i, device := range model.devices {
		// Ensure all devices are unpaused.
		err := device.Unpause(manual)
		if err != nil && err != DeviceAlreadyPaused {
			for i -= 1; i >= 0; i -= 1 {
				device.Pause(manual)
			}
			return err
		}
	}

	// All good.
	return nil
}

func (model *Model) Load(vm *platform.Vm) error {

	for _, device := range model.devices {
		// Load our device state.
		err := device.Load(vm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (model *Model) Save(vm *platform.Vm) error {

	for _, device := range model.devices {
		// Synchronize our device state.
		err := device.Save(vm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (model *Model) DeviceInfo(vm *platform.Vm) ([]DeviceInfo, error) {

	err := model.Pause(false)
	if err != nil {
		return nil, err
	}
	defer model.Unpause(false)

	// Synchronize our state.
	err = model.Save(vm)
	if err != nil {
		return nil, err
	}

	devices := make([]DeviceInfo, 0, len(model.devices))
	for _, device := range model.devices {

		// Get the deviceinfo.
		deviceinfo, err := NewDeviceInfo(device)
		if err != nil {
			return nil, err
		}

		devices = append(devices, deviceinfo)
	}

	return devices, nil
}
