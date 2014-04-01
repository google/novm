package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int IoctlGetClock = KVM_GET_CLOCK;
const int IoctlSetClock = KVM_SET_CLOCK;
*/
import "C"

import (
    "syscall"
    "unsafe"
)

//
// Our clock state.
//
type Clock struct {
    Time  uint64 `json:"time"`
    Flags uint32 `json:"flags"`
}

func (vm *Vm) GetClock() (Clock, error) {

    // Execute the ioctl.
    var kvm_clock_data C.struct_kvm_clock_data
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.IoctlGetClock),
        uintptr(unsafe.Pointer(&kvm_clock_data)))
    if e != 0 {
        return Clock{}, e
    }

    return Clock{
        Time:  uint64(kvm_clock_data.clock),
        Flags: uint32(kvm_clock_data.flags),
    }, nil
}

func (vm *Vm) SetClock(clock Clock) error {

    // Execute the ioctl.
    var kvm_clock_data C.struct_kvm_clock_data
    kvm_clock_data.clock = C.__u64(clock.Time)
    kvm_clock_data.flags = C.__u32(clock.Flags)
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.IoctlSetClock),
        uintptr(unsafe.Pointer(&kvm_clock_data)))
    if e != 0 {
        return e
    }

    return nil
}
