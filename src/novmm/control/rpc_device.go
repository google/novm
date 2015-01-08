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

package control

import (
	"regexp"
)

//
// Low-level device controls.
//

type DeviceSettings struct {
	// Name.
	Name string `json:"name"`

	// Drvier.
	Driver string `json:"driver"`

	// Debug?
	Debug bool `json:"debug"`

	// Pause?
	Paused bool `json:"paused"`
}

func (rpc *Rpc) Device(settings *DeviceSettings, nop *Nop) error {

	rn, err := regexp.Compile(settings.Name)
	if err != nil {
		return err
	}

	rd, err := regexp.Compile(settings.Driver)
	if err != nil {
		return err
	}

	for _, device := range rpc.model.Devices() {

		if rn.MatchString(device.Name()) &&
			rd.MatchString(device.Driver()) {

			device.SetDebugging(settings.Debug)

			if settings.Paused {
				err = device.Pause(true)
			} else {
				err = device.Unpause(true)
			}

			if err != nil {
				break
			}
		}
	}

	return err
}
