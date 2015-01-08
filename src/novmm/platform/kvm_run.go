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

package platform

/*
#include "kvm_run.h"
*/
import "C"

import (
	"novmm/utils"
	"sync"
	"syscall"
)

type RunInfo struct {
	// C-level nformation.
	info C.struct_kvm_run_info

	// Are we running?
	is_running bool

	// Are we paused manually?
	is_paused bool

	// Our internal pause count.
	// This is for internal services that may need
	// to pause vcpus (such as suspend/resume), and
	// is tracked independently of the manual pause.
	paused int

	// Our run lock.
	lock         *sync.Mutex
	pause_event  *sync.Cond
	resume_event *sync.Cond
}

func (vcpu *Vcpu) initRunInfo() error {
	// Initialize our structure.
	e := syscall.Errno(C.kvm_run_init(C.int(vcpu.fd), &vcpu.RunInfo.info))
	if e != 0 {
		return e
	}

	// Setup the lock.
	vcpu.RunInfo.lock = &sync.Mutex{}
	vcpu.RunInfo.pause_event = sync.NewCond(vcpu.RunInfo.lock)
	vcpu.RunInfo.resume_event = sync.NewCond(vcpu.RunInfo.lock)

	// We're okay.
	return nil
}

func (vcpu *Vcpu) Run() error {
	for {
		// Make sure our registers are flushed.
		// This will also refresh registers after we
		// execute but are interrupted (i.e. EINTR).
		err := vcpu.flushAllRegs()
		if err != nil {
			return err
		}

		// Ensure we can run.
		//
		// For exact semantics, see Pause() and Unpause().
		// NOTE: By default, we are always "running". We are
		// only not running when we arrive at this point in
		// the pipeline and are waiting on the resume_event.
		//
		// This is because we want to ensure that our registers
		// have been flushed and all that devices are up-to-date
		// before we can declare a VCPU as "paused".
		vcpu.RunInfo.lock.Lock()

		for vcpu.RunInfo.is_paused || vcpu.RunInfo.paused > 0 {
			// Note that we are not running,
			// See NOTE above about what this means.
			vcpu.RunInfo.is_running = false

			// Send a notification that we are paused.
			vcpu.RunInfo.pause_event.Broadcast()

			// Wait for a wakeup notification.
			vcpu.RunInfo.resume_event.Wait()
		}

		vcpu.RunInfo.is_running = true
		vcpu.RunInfo.lock.Unlock()

		// Execute our run ioctl.
		rc := C.kvm_run(
			C.int(vcpu.fd),
			C.int(utils.SigVcpuInt),
			&vcpu.RunInfo.info)
		e := syscall.Errno(rc)

		if e == syscall.EINTR || e == syscall.EAGAIN {
			continue
		} else if e != 0 {
			return e
		} else {
			break
		}
	}

	return vcpu.GetExitError()
}

func (vcpu *Vcpu) Pause(manual bool) error {
	// Acquire our runlock.
	vcpu.RunInfo.lock.Lock()
	defer vcpu.RunInfo.lock.Unlock()

	if manual {
		// Already paused?
		if vcpu.RunInfo.is_paused {
			return AlreadyPaused
		}
		vcpu.RunInfo.is_paused = true
	} else {
		// Bump our pause count.
		vcpu.RunInfo.paused += 1
	}

	// Are we running? Need to interrupt.
	// We don't return from this function (even if there
	// are multiple callers) until we are sure that the VCPU
	// is actually paused, and all devices are up-to-date.
	if vcpu.is_running {
		// Only the first caller need interrupt.
		if manual || vcpu.RunInfo.paused == 1 {
			e := C.kvm_run_interrupt(
				C.int(vcpu.fd),
				C.int(utils.SigVcpuInt),
				&vcpu.RunInfo.info)
			if e != 0 {
				return syscall.Errno(e)
			}
		}

		// Wait for the vcpu to notify that it is paused.
		vcpu.RunInfo.pause_event.Wait()
	}

	return nil
}

func (vcpu *Vcpu) Unpause(manual bool) error {
	// Acquire our runlock.
	vcpu.RunInfo.lock.Lock()
	defer vcpu.RunInfo.lock.Unlock()

	// Are we actually paused?
	// This was not a valid call.
	if manual {
		// Already unpaused?
		if !vcpu.RunInfo.is_paused {
			return NotPaused
		}
		vcpu.RunInfo.is_paused = false
	} else {
		if vcpu.RunInfo.paused == 0 {
			return NotPaused
		}

		// Decrease our pause count.
		vcpu.RunInfo.paused -= 1
	}

	// Are we still paused?
	if vcpu.RunInfo.is_paused || vcpu.RunInfo.paused > 0 {
		return nil
	}

	// Allow the vcpu to resume.
	vcpu.RunInfo.resume_event.Broadcast()

	return nil
}
