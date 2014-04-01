package machine

import (
    "novmm/platform"
)

type Apic struct {
    BaseDevice

    // Our addresses.
    // At the moment, these are at fixed address.
    // But just check that they meet expectations.
    IOApic platform.Paddr `json:"ioapic"`
    LApic  platform.Paddr `json:"lapic"`

    // Our platform APIC.
    State platform.IrqChip `json:"state"`
}

func NewApic(info *DeviceInfo) (Device, error) {

    apic := new(Apic)

    // Figure out our Apic addresses.
    // See the note above re: fixed addresses.
    apic.IOApic = platform.IOApic()
    apic.LApic = platform.LApic()

    return apic, apic.init(info)
}

func (apic *Apic) Attach(vm *platform.Vm, model *Model) error {

    // Reserve our IOApic and LApic.
    err := model.Reserve(
        vm,
        apic,
        MemoryTypeReserved,
        apic.IOApic,
        platform.PageSize,
        nil)
    if err != nil {
        return err
    }
    err = model.Reserve(
        vm,
        apic,
        MemoryTypeReserved,
        apic.LApic,
        platform.PageSize,
        nil)
    if err != nil {
        return err
    }

    // Create our irqchip.
    err = vm.CreateIrqChip()
    if err != nil {
        return err
    }

    // We're good.
    return nil
}

func (apic *Apic) Save(vm *platform.Vm) error {

    var err error

    // Save our IrqChip state.
    apic.State, err = vm.GetIrqChip()
    if err != nil {
        return err
    }

    // We're good.
    return nil
}

func (apic *Apic) Load(vm *platform.Vm) error {
    // Load state.
    return vm.SetIrqChip(apic.State)
}
