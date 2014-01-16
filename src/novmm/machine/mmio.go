package machine

import (
    "novmm/platform"
)

type MmioEvent struct {
    *platform.ExitMmio
}

func (mmio MmioEvent) Size() uint {
    return mmio.ExitMmio.Length()
}

func (mmio MmioEvent) GetData() uint64 {
    return *mmio.ExitMmio.Data()
}

func (mmio MmioEvent) SetData(val uint64) {
    *mmio.ExitMmio.Data() = val
}

func (mmio MmioEvent) IsWrite() bool {
    return mmio.ExitMmio.IsWrite()
}

type MmioDevice struct {
    BaseDevice

    // A map of our available I/O.
    IoMap      `json:"-"`
    IoHandlers `json:"-"`

    // Our address in memory.
    Offset platform.Paddr `json:"address"`

    // Our assigned interrupt.
    InterruptNumber platform.Irq `json:"interrupt"`

    // Regions that should be reserved.
    // NOTE: These have the offset applied.
    reservations []MemoryRegion `json:"-"`
}

func (mmio *MmioDevice) MmioHandlers() IoHandlers {
    return mmio.IoHandlers
}

func (mmio *MmioDevice) Attach(vm *platform.Vm, model *Model) error {

    // Build our IO Handlers.
    mmio.IoHandlers = make(IoHandlers)
    for region, ops := range mmio.IoMap {
        new_region := MemoryRegion{region.Start + mmio.Offset, region.Size}
        mmio.IoHandlers[new_region] = NewIoHandler(mmio, new_region.Start, ops)
    }

    // Reserve memory regions.
    if mmio.reservations != nil {
        for _, region := range mmio.reservations {
            err := model.Reserve(
                vm,
                mmio,
                MemoryTypeReserved,
                region.Start+mmio.Offset,
                region.Size,
                nil)
            if err != nil {
                return err
            }
        }
    }

    if mmio.InterruptNumber != 0 {
        // Reserve our interrupt.
        _, ok := model.InterruptMap[mmio.InterruptNumber]
        if ok {
            // Already a device there.
            return InterruptConflict
        }
        model.InterruptMap[mmio.InterruptNumber] = mmio

    } else {
        // Find an interrupt.
        for irq := platform.Irq(16); ; irq += 1 {
            if _, ok := model.InterruptMap[irq]; !ok {
                model.InterruptMap[irq] = mmio
                mmio.InterruptNumber = irq
            }
        }
    }

    return mmio.BaseDevice.Attach(vm, model)
}
