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

package main

import (
	"log"
	"novmm/loader"
	"novmm/machine"
	"novmm/platform"
	"runtime"
)

func Loop(
	vm *platform.Vm,
	vcpu *platform.Vcpu,
	model *machine.Model,
	tracer *loader.Tracer) error {

	// It's not really kosher to switch threads constantly when running a
	// KVM VCPU. So we simply lock this goroutine to a single system
	// thread. That way we know it won't be bouncing around.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Printf("Vcpu[%d] running.", vcpu.Id)

	for {
		// Enter the guest.
		err := vcpu.Run()

		// Trace if requested.
		trace_err := tracer.Trace(vcpu, vcpu.IsStepping())
		if trace_err != nil {
			return trace_err
		}

		// No reason for exit?
		if err == nil {
			return ExitWithoutReason
		}

		// Handle the error.
		switch err.(type) {
		case *platform.ExitPio:
			err = model.HandlePio(vm, err.(*platform.ExitPio))

		case *platform.ExitMmio:
			err = model.HandleMmio(vm, err.(*platform.ExitMmio))

		case *platform.ExitDebug:
			err = nil

		case *platform.ExitShutdown:
			// Vcpu shutdown.
			return nil
		}

		// Error handling the exit.
		if err != nil {
			return err
		}
	}

	// Unreachable.
	return nil
}
