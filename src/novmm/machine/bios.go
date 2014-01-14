package machine

import (
    "novmm/platform"
)

type Bios struct {
    BaseDevice

    // Our reserved high-mem.
    Reserved platform.Paddr `json:"reserved"`
    TSSAddr  platform.Paddr `json:"tss"`
}

func NewBios(info *DeviceInfo) (Device, error) {
    bios := new(Bios)
    bios.Reserved = 0xf0000000
    bios.TSSAddr = 0xfffbc000
    return bios, bios.Init(info)
}

func (bios *Bios) Attach(vm *platform.Vm, model *Model) error {

    // Reserve our basic "BIOS" memory.
    // This is done simply to match expectations.
    err := model.Reserve(
        vm,
        bios,
        MemoryTypeReserved,
        platform.Paddr(0), // Start.
        platform.PageSize, // Size.
        nil)
    if err != nil {
        return err
    }

    ioapic := vm.IOApic()
    lapic := vm.LApic()
    var minpic platform.Paddr
    var maxpic platform.Paddr
    if ioapic < lapic {
        minpic = ioapic
        maxpic = lapic
    } else {
        minpic = lapic
        maxpic = ioapic
    }

    // This is so operating system is able to map
    // pci BARs within a 32-bit range. This is also
    // necessary because the LAPICs and IOAPICs are
    // mapped here, and it should be reserved.
    err = model.Reserve(
        vm,
        bios,
        MemoryTypeReserved,
        bios.Reserved,
        uint64(minpic-bios.Reserved),
        nil)
    if err != nil {
        return err
    }

    // Reserve our IOApic and LApic.
    err = model.Reserve(
        vm,
        bios,
        MemoryTypeReserved,
        minpic,
        platform.PageSize,
        nil)
    if err != nil {
        return err
    }
    err = model.Reserve(
        vm,
        bios,
        MemoryTypeReserved,
        maxpic,
        platform.PageSize,
        nil)
    if err != nil {
        return err
    }

    // Now reserve our TSS.
    err = model.Reserve(
        vm,
        bios,
        MemoryTypeSpecial,
        bios.TSSAddr,
        vm.SizeSpecialMemory(),
        nil)
    if err != nil {
        return err
    }

    // Finish the region.
    tss_end := bios.TSSAddr.After(vm.SizeSpecialMemory())
    return model.Reserve(
        vm,
        bios,
        MemoryTypeReserved,
        tss_end,
        uint64(platform.Paddr(0x100000000)-tss_end),
        nil)
}
