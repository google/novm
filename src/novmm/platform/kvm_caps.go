// +build linux
package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int CheckExtension = KVM_CHECK_EXTENSION;

// Capabilities (extensions).
const int CapUserMem = KVM_CAP_USER_MEMORY;
const int CapIrqChip = KVM_CAP_IRQCHIP;
const int CapIoFd = KVM_CAP_IOEVENTFD;
const int CapIrqFd = KVM_CAP_IRQFD;
const int CapPit2 = KVM_CAP_PIT2;
const int CapPitState2 = KVM_CAP_PIT_STATE2;
const int CapGuestDebug = KVM_CAP_SET_GUEST_DEBUG;
const int CapCpuid = KVM_CAP_EXT_CPUID;
const int CapSignalMsi = KVM_CAP_SIGNAL_MSI;

// NOTE: Not really generally available yet.
// This is a pretty new feature, but once it's available
// it surely will allow rearchitecting some of the MMIO-based
// devices to operate more efficently (as the guest will only
// trap out on WRITEs, and not on READs).
// const int MemReadOnly = KVM_MEM_READONLY;
// const int CapReadOnlyMem = KVM_CAP_READONLY_MEM;
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

var requiredCapabilities = []kvmCapability{
    kvmCapability{"User Memory", uintptr(C.CapUserMem)},
    kvmCapability{"IRQ Chip", uintptr(C.CapIrqChip)},
    kvmCapability{"IO Event FD", uintptr(C.CapIoFd)},
    kvmCapability{"IRQ Event FD", uintptr(C.CapIrqFd)},
    kvmCapability{"PIT2", uintptr(C.CapPit2)},
    kvmCapability{"PITSTATE2", uintptr(C.CapPitState2)},
    kvmCapability{"CPUID", uintptr(C.CapCpuid)},
    kvmCapability{"MSI", uintptr(C.CapSignalMsi)},

    // It does seem to be the case that this capability
    // is not advertised correctly. On my kernel (3.11),
    // it supports this ioctl but yet claims this capability
    // is not available.
    // In any case, this isn't necessary functionality,
    // but the call to SetSingleStep() may fail.
    // kvmCapability{"Guest debug", uintptr(C.CapGuestDebug)},

    // See NOTE above.
    // kvmCapability{"Read-only Memory", uintptr(C.CapReadOnlyMem)},
}

func checkCapability(
    fd int,
    capability kvmCapability) error {

    r, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(C.CheckExtension),
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
