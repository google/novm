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
	"novmm/loader"
	"novmm/machine"
	"novmm/platform"
)

//
// Rpc --
//
// This is basic state provided to the
// Rpc interface. All Rpc functions have
// access to this state (but nothing else).
//

type Rpc struct {
	// Our device model.
	model *machine.Model

	// Our underlying Vm object.
	vm *platform.Vm

	// Our tracer.
	tracer *loader.Tracer
}

func NewRpc(
	model *machine.Model,
	vm *platform.Vm,
	tracer *loader.Tracer) *Rpc {

	return &Rpc{
		model:  model,
		vm:     vm,
		tracer: tracer,
	}
}

//
// The Noop --
//
// Many of our operations do not require
// a specific parameter or a specific return.
//
type Nop struct{}
