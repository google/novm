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

type Pit struct {
	BaseDevice

	// Our pit state.
	// Similar to the pit, we consider the platform
	// PIT to be an intrinsic part of our "pit".
	Pit platform.PitState `json:"pit"`
}

func NewPit(info *DeviceInfo) (Device, error) {
	pit := new(Pit)
	return pit, pit.init(info)
}

func (pit *Pit) Attach(vm *platform.Vm, model *Model) error {

	// Create our PIT.
	err := vm.CreatePit()
	if err != nil {
		return err
	}

	// We're good.
	return nil
}

func (pit *Pit) Save(vm *platform.Vm) error {

	var err error

	// Save our Pit state.
	pit.Pit, err = vm.GetPit()
	if err != nil {
		return err
	}

	// We're good.
	return nil
}

func (pit *Pit) Load(vm *platform.Vm) error {
	// Load state.
	return vm.SetPit(pit.Pit)
}
