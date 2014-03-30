package control

import (
    "syscall"
)

//
// Low-level vcpu controls.
//

type VcpuSettings struct {
    // Which vcpu?
    Id  int `json:"id"`

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
