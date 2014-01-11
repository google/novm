package machine

import (
    "novmm/platform"
)

type Bios struct {
    BaseDevice
}

func NewBios(info *DeviceInfo) (Device, error) {
    bios := new(Bios)
    return bios, bios.Init(info)
}

func (bios *Bios) Attach(vm *platform.Vm, model *Model) error {

    // Reserve our basic "BIOS" memory.
    // This is done simply to match expectations.
    return model.Reserve(
        vm,
        bios,
        MemoryTypeReserved,
        platform.Paddr(0), // Start.
        platform.PageSize, // Size.
        nil)
}
