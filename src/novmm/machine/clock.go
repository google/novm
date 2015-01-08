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

type Clock struct {
	BaseDevice

	// Our clock state.
	Clock platform.Clock `json:"clock"`
}

func NewClock(info *DeviceInfo) (Device, error) {
	clock := new(Clock)
	return clock, clock.init(info)
}

func (clock *Clock) Attach(vm *platform.Vm, model *Model) error {
	// We're good.
	return nil
}

func (clock *Clock) Save(vm *platform.Vm) error {

	var err error

	// Save our clock state.
	clock.Clock, err = vm.GetClock()
	if err != nil {
		return err
	}

	// We're good.
	return nil
}

func (clock *Clock) Load(vm *platform.Vm) error {
	// Load state.
	return vm.SetClock(clock.Clock)
}
