package machine

import (
    "novmm/platform"
)

//
// InterruptMap --
//
// Interrupts are much simpler than our
// memory layout. We simply store a map
// of allocated interrupts with a pointer
// to the device info.

type InterruptMap map[uint]*InterruptDevice

type InterruptDevice struct {
    Interrupt uint `json:"interrupt"`
}

func (idev *InterruptDevice) AttachInterrupt(vm *platform.Vm, model *Model) error {

    if idev.Interrupt != 0 {
        // Reserve our interrupt.
        _, ok := model.InterruptMap[idev.Interrupt]
        if ok {
            // Already a device there.
            return InterruptConflict
        }
        model.InterruptMap[idev.Interrupt] = idev
        return nil
    }

    // Find an interrupt.
    for irq := uint(16); irq < 24; irq += 1 {
        if _, ok := model.InterruptMap[irq]; !ok {
            model.InterruptMap[irq] = idev
            idev.Interrupt = irq
            return nil
        }
    }

    // Nothing available?
    return InterruptUnavailable
}
