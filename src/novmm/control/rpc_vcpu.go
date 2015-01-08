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
	"syscall"
)

//
// Low-level vcpu controls.
//

type VcpuSettings struct {
	// Which vcpu?
	Id int `json:"id"`

	// Single stepping?
	Step bool `json:"step"`

	// Paused?
	Paused bool `json:"paused"`
}

func (rpc *Rpc) Vcpu(settings *VcpuSettings, nop *Nop) error {
	// A valid vcpu?
	vcpus := rpc.vm.Vcpus()
	if settings.Id >= len(vcpus) {
		return syscall.EINVAL
	}

	// Grab our specific vcpu.
	vcpu := vcpus[settings.Id]

	// Ensure steping is as expected.
	err := vcpu.SetStepping(settings.Step)
	if err != nil {
		return err
	}

	// Ensure that the vcpu is paused/unpaused.
	if settings.Paused {
		err = vcpu.Pause(true)
	} else {
		err = vcpu.Unpause(true)
	}

	// Done.
	return err
}
