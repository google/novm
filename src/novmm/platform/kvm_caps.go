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

// +build linux
package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int IoctlCheckExtension = KVM_CHECK_EXTENSION;

// Capabilities (extensions).
const int CapUserMem = KVM_CAP_USER_MEMORY;
const int CapSetIdentityMapAddr = KVM_CAP_SET_IDENTITY_MAP_ADDR;
const int CapIrqChip = KVM_CAP_IRQCHIP;
const int CapIoFd = KVM_CAP_IOEVENTFD;
const int CapIrqFd = KVM_CAP_IRQFD;
const int CapPit2 = KVM_CAP_PIT2;
const int CapPitState2 = KVM_CAP_PIT_STATE2;
const int CapCpuid = KVM_CAP_EXT_CPUID;
const int CapSignalMsi = KVM_CAP_SIGNAL_MSI;
const int CapVcpuEvents = KVM_CAP_VCPU_EVENTS;
const int CapAdjustClock = KVM_CAP_ADJUST_CLOCK;
const int CapXSave = KVM_CAP_XSAVE;
const int CapXcrs = KVM_CAP_XCRS;
*/
import "C"

import (
	"syscall"
)

type kvmCapability struct {
	name   string
	number uintptr
}

func (capability *kvmCapability) Error() string {
	return "Missing capability: " + capability.name
}

//
// Our required capabilities.
//
// Many of these are actually optional, but none
// of the plumbing has been done to gracefully fail
// when they are not available. For the time being
// development is focused on legacy-free environments,
// so we can split this out when it's necessary later.
//
var requiredCapabilities = []kvmCapability{
	kvmCapability{"User Memory", uintptr(C.CapUserMem)},
	kvmCapability{"Identity Map", uintptr(C.CapSetIdentityMapAddr)},
	kvmCapability{"IRQ Chip", uintptr(C.CapIrqChip)},
	kvmCapability{"IO Event FD", uintptr(C.CapIoFd)},
	kvmCapability{"IRQ Event FD", uintptr(C.CapIrqFd)},
	kvmCapability{"PIT2", uintptr(C.CapPit2)},
	kvmCapability{"PITSTATE2", uintptr(C.CapPitState2)},
	kvmCapability{"Clock", uintptr(C.CapAdjustClock)},
	kvmCapability{"CPUID", uintptr(C.CapCpuid)},
	kvmCapability{"MSI", uintptr(C.CapSignalMsi)},
	kvmCapability{"VCPU Events", uintptr(C.CapVcpuEvents)},
	kvmCapability{"XSAVE", uintptr(C.CapXSave)},
	kvmCapability{"XCRS", uintptr(C.CapXcrs)},
}

func checkCapability(
	fd int,
	capability kvmCapability) error {

	r, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(C.IoctlCheckExtension),
		capability.number)
	if r != 1 || e != 0 {
		return &capability
	}

	return nil
}

func checkCapabilities(fd int) error {
	// Check our extensions.
	for _, capSpec := range requiredCapabilities {
		err := checkCapability(fd, capSpec)
		if err != nil {
			return err
		}
	}

	return nil
}
