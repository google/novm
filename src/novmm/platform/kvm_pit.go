// +build linux
package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int CreatePit2 = KVM_CREATE_PIT2;
const int GetPit2 = KVM_GET_PIT2;
const int SetPit2 = KVM_SET_PIT2;

// Size of pit state.
const int PitSize = sizeof(struct kvm_pit_state2);
*/
import "C"

import (
    "log"
    "syscall"
    "unsafe"
)

//
// PitState --
//
// We represent the PitState as a blob.
// This representation should be relatively
// safe from a forward-compatibility perspective,
// as KVM internally will take care of reserving
// bits and ensuring compatibility, etc.
//
type PitState struct {
    Data []byte `json:"data"`
}

func (vm *Vm) CreatePit() error {
    // Prepare the PIT config.
    // The only flag supported at the time of writing
    // was KVM_PIT_SPEAKER_DUMMY, which I really have no
    // interest in supporting.
    var pit C.struct_kvm_pit_config
    pit.flags = C.__u32(0)

    // Execute the ioctl.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.CreatePit2),
        uintptr(unsafe.Pointer(&pit)))
    if e != 0 {
        return e
    }

    log.Print("kvm: PIT created.")
    return nil
}

func (vm *Vm) GetPit() (PitState, error) {

    // Prepare the pit state.
    state := PitState{make([]byte, C.PitSize, C.PitSize)}

    // Execute the ioctl.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.GetPit2),
        uintptr(unsafe.Pointer(&state.Data[0])))
    if e != 0 {
        return state, e
    }

    return state, nil
}

func (vm *Vm) SetPit(state PitState) error {

    // Is there any state to set?
    // We just eat this error, it's fine.
    if state.Data == nil {
        return nil
    }

    // Is this the right size?
    if len(state.Data) != int(C.PitSize) {
        return PitIncompatible
    }

    // Execute the ioctl.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.SetPit2),
        uintptr(unsafe.Pointer(&state.Data[0])))
    if e != 0 {
        return e
    }

    return nil
}
