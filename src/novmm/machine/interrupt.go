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

type InterruptMap map[platform.Irq]Device
