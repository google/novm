package control

import (
    "novmm/platform"
)

//
// High-level controls.
//

func (control *Control) Pause(nopin *Nop, nopout *Nop) error {
    // Get all vcpus.
    vcpus := control.vm.Vcpus()

    // Pause them all.
    for _, vcpu := range vcpus {
        err := vcpu.Pause(true)
        if err != nil && err != platform.AlreadyPaused {
            return err
        }
    }

    // Done.
    return nil
}

func (control *Control) Unpause(nopin *Nop, nopout *Nop) error {
    // Get all vcpus.
    vcpus := control.vm.Vcpus()

    // Unpause them all.
    for _, vcpu := range vcpus {
        err := vcpu.Unpause(true)
        if err != nil && err != platform.NotPaused {
            return err
        }
    }

    // Done.
    return nil
}
