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
	"log"
	"novmm/platform"
	"sync"
)

type IoMap map[MemoryRegion]IoOperations
type IoHandlers map[MemoryRegion]*IoHandler

type BaseDevice struct {
	// Pointer to original device info.
	// This is reference in serialization.
	// (But is explicitly not exported, as
	// the device info will have a reference
	// back to this new device object).
	info *DeviceInfo

	// Have we been paused manually?
	is_paused bool

	// Internal pause count.
	paused int

	// Our internal lock for pause/resume.
	// This is significantly simpler than the
	// VCPU case, so we can get away with using
	// just a straight-forward RWMUtex.
	pause_lock sync.Mutex
	run_lock   sync.RWMutex
}

type Device interface {
	Name() string
	Driver() string

	PioHandlers() IoHandlers
	MmioHandlers() IoHandlers

	Attach(vm *platform.Vm, model *Model) error
	Load(vm *platform.Vm) error
	Save(vm *platform.Vm) error

	Pause(manual bool) error
	Unpause(manual bool) error

	Acquire()
	Release()

	Interrupt() error

	Debug(format string, v ...interface{})
	IsDebugging() bool
	SetDebugging(debug bool)
}

func (device *BaseDevice) init(info *DeviceInfo) error {
	// Save our original device info.
	// This isn't structural (hence no export).
	device.info = info
	return nil
}

func (device *BaseDevice) Name() string {
	return device.info.Name
}

func (device *BaseDevice) Driver() string {
	return device.info.Driver
}

func (device *BaseDevice) PioHandlers() IoHandlers {
	return IoHandlers{}
}

func (device *BaseDevice) MmioHandlers() IoHandlers {
	return IoHandlers{}
}

func (device *BaseDevice) Attach(vm *platform.Vm, model *Model) error {
	return nil
}

func (device *BaseDevice) Load(vm *platform.Vm) error {
	return nil
}

func (device *BaseDevice) Save(vm *platform.Vm) error {
	return nil
}

func (device *BaseDevice) Pause(manual bool) error {
	device.pause_lock.Lock()
	defer device.pause_lock.Unlock()

	if manual {
		if device.is_paused {
			return DeviceAlreadyPaused
		}
		device.is_paused = true
		if device.paused > 0 {
			// Already paused.
			return nil
		}
	} else {
		device.paused += 1
		if device.paused > 1 || device.is_paused {
			// Already paused.
			device.paused += 1
			return nil
		}
	}

	// Acquire our runlock, preventing
	// any execution from continuing.
	device.run_lock.Lock()
	return nil
}

func (device *BaseDevice) Unpause(manual bool) error {
	device.pause_lock.Lock()
	defer device.pause_lock.Unlock()

	if manual {
		if !device.is_paused {
			return DeviceNotPaused
		}
		device.is_paused = false
		if device.paused > 0 {
			// Please don't unpause.
			return nil
		}
	} else {
		device.paused -= 1
		if device.paused > 0 || device.is_paused {
			// Please don't unpause.
			return nil
		}
	}

	// Release our runlock, allow
	// execution to continue normally.
	device.run_lock.Unlock()
	return nil
}

func (device *BaseDevice) Acquire() {
	device.run_lock.RLock()
}

func (device *BaseDevice) Release() {
	device.run_lock.RUnlock()
}

func (device *BaseDevice) Interrupt() error {
	return nil
}

func (device *BaseDevice) Debug(format string, v ...interface{}) {
	if device.IsDebugging() {
		log.Printf(device.Name()+": "+format, v...)
	}
}

func (device *BaseDevice) IsDebugging() bool {
	return device.info.Debug
}

func (device *BaseDevice) SetDebugging(debug bool) {
	device.info.Debug = debug
}
