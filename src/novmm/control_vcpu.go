package main

import (
    "syscall"
)

type VcpuSettings struct {
    // Which vcpu?
    Id  int `json:"id"`

    // Single stepping?
    Step bool `json:"step"`
}

func (control *Control) Vcpu(settings *VcpuSettings, ok *bool) error {
    // A valid vcpu?
    vcpus := control.vm.GetVcpus()
    if settings.Id >= len(vcpus) {
        *ok = false
        return syscall.EINVAL
    }
    vcpu := vcpus[settings.Id]
    err := vcpu.SetStepping(settings.Step)
    *ok = (err == nil)
    return err
}
