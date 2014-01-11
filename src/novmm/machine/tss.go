package machine

import (
    "novmm/platform"
)

type Tss struct {
    BaseDevice

    // Our address.
    Addr platform.Paddr `json:"address"`
}

func NewTss(info *DeviceInfo) (Device, error) {
    tss := new(Tss)
    tss.Addr = 0xfffbc000 // Sensible default.
    return tss, tss.Init(info)
}

func (tss *Tss) Attach(vm *platform.Vm, model *Model) error {

    // Reserve a block of memory for internal use.
    return model.Reserve(
        vm,
        tss,
        MemoryTypeSpecial,
        tss.Addr,               // Start.
        vm.SizeSpecialMemory(), // Size.
        nil)
}
