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

//
// State-related rpcs.
//

func (rpc *Rpc) State(nop *Nop, res *State) error {

	state, err := SaveState(rpc.vm, rpc.model)
	if err != nil {
		return err
	}

	// Save our state.
	res.Vcpus = state.Vcpus
	res.Devices = state.Devices

	return err
}

func (rpc *Rpc) Reload(in *Nop, out *Nop) error {

	// Pause the vm.
	// This is kept pausing for the entire reload().
	err := rpc.vm.Pause(false)
	if err != nil {
		return err
	}
	defer rpc.vm.Unpause(false)

	// Save a copy of the current state.
	state, err := SaveState(rpc.vm, rpc.model)
	if err != nil {
		return err
	}

	// Reload all vcpus.
	for i, vcpuspec := range state.Vcpus {
		err := rpc.vm.Vcpus()[i].Load(vcpuspec)
		if err != nil {
			return err
		}
	}

	// Reload all device state.
	return rpc.model.Load(rpc.vm)
}
