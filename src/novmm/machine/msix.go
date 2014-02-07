package machine

import (
    "math"
    "novmm/platform"
)

const (
    PciCapabilityMSIX = 0x11
)

type MsiXConf struct {
    *MsiXDevice `json:"-"`

    // The MSI control register.
    Control Register `json:"control"`

    // The table offset (& BAR).
    TableOffset Register `json:"table"`

    // The PBA offset (& BAR).
    PbaOffset Register `json:"pba"`
}

type MsiXEntry struct {
    *MsiXDevice `json:"-"`

    // The control dword.
    Control Register `json:"control"`

    // The lower address (+masked & pending bits).
    Address Register `json:"address"`

    // The Data.
    Data Register `json:"data"`
}

type MsiXDevice struct {
    *PciDevice

    // This is a pointer to the pci capability.
    // NOTE: We add it and rediscover it in order
    // to avoid serialization problems.
    conf *MsiXConf

    // Our saved interrupt function.
    msi_interrupt func(addr platform.Paddr, data uint32) error

    // Our pending bit array.
    Pending Ram `json:"pending"`

    // The entries are a device that we expose
    // to the PCI Bar as specified in the creation.
    Entries []MsiXEntry `json:"entries"`
}

func (msix *MsiXEntry) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case 0:
        return msix.Address.Read(0, size)
    case 4:
        return msix.Data.Read(0, size)
    case 6:
        return msix.Control.Read(0, size)
    }

    return math.MaxUint64, nil
}

func (msix *MsiXEntry) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case 0:
        return msix.Address.Write(0, size, value)
    case 4:
        return msix.Data.Write(0, size, value)
    case 6:
        return msix.Control.Write(0, size, value)
    }

    return nil
}

func (msix *MsiXDevice) IsPending(vector int) bool {
    return msix.Pending.Get8(int(vector/8))&(1<<uint(vector%8)) != 0
}

func (msix *MsiXDevice) SetPending(vector int) {
    val := msix.Pending.Get8(int(vector / 8))
    msix.Pending.Set8(int(vector/8), val|byte(1<<uint(vector%8)))
}

func (msix *MsiXDevice) ClearPending(vector int) {
    val := msix.Pending.Get8(int(vector / 8))
    msix.Pending.Set8(int(vector/8), val & ^byte(1<<uint(vector%8)))
}

func (msix *MsiXDevice) IsMasked(vector int) bool {
    if msix.conf.Control.Value&0x40 != 0 {
        return true
    }
    entry := msix.FindEntry(int(vector / 8))
    if entry != nil && entry.Control.Value&0x1 != 0 {
        return true
    }
    return false
}

func (msix *MsiXDevice) CheckPending(vector int) {
    if msix.IsPending(vector) && !msix.IsMasked(vector) {
        msix.SendInterrupt(vector)
    }
}

func (msix *MsiXDevice) CheckAllPending() {
    for i, _ := range msix.Entries {
        msix.CheckPending(i)
    }
}

func (msix *MsiXDevice) FindEntry(vector int) *MsiXEntry {
    if vector >= len(msix.Entries) {
        return nil
    }

    return &msix.Entries[vector]
}

func (msix *MsiXDevice) Read(offset uint64, size uint) (uint64, error) {

    if offset < uint64(msix.Pending.Size()) {
        return msix.Pending.Read(offset, size)
    }
    offset -= uint64(msix.Pending.Size())

    entry := msix.FindEntry(int(offset / 8))
    if entry == nil {
        return math.MaxUint64, nil
    }

    return entry.Read(offset%8, size)
}

func (msix *MsiXDevice) Write(offset uint64, size uint, value uint64) error {

    if offset < uint64(msix.Pending.Size()) {
        return msix.Pending.Write(offset, size, value)
    }
    offset -= uint64(msix.Pending.Size())

    entry := msix.FindEntry(int(offset / 8))
    if entry == nil {
        return nil
    }

    defer msix.CheckPending(int(offset / 8))
    return entry.Write(offset%8, size, value)
}

func (msix *MsiXConf) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case 0:
        return msix.Control.Read(0, size)
    case 2:
        return msix.TableOffset.Read(0, size)
    case 6:
        return msix.PbaOffset.Read(0, size)
    }

    return math.MaxUint64, nil
}

func (msix *MsiXConf) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case 0:
        defer msix.MsiXDevice.CheckAllPending()
        return msix.Control.Write(0, size, value)
    case 2:
        return msix.TableOffset.Write(0, size, value)
    case 6:
        return msix.PbaOffset.Write(0, size, value)
    }

    return nil
}

func NewMsiXDevice(
    pcidevice *PciDevice,
    barno uint,
    vectors uint) *MsiXDevice {

    msix := new(MsiXDevice)
    msix.PciDevice = pcidevice
    msix.Entries = make([]MsiXEntry, vectors, vectors)

    // Initialize our entries.
    for _, entry := range msix.Entries {
        entry.Control.Value = 0x1 // Masked.
    }

    // Create our new pending bit array.
    pending_size := 16 * (vectors + 63) / 64
    msix.Pending = make(Ram, 0, 0)
    msix.Pending.GrowTo(int(pending_size))

    // Create a new set of control registers.
    msix.conf = new(MsiXConf)
    msix.conf.Control.readonly = 0xffffffffffffff7f
    msix.conf.Control.Value = uint64(vectors - 1)
    msix.conf.TableOffset.readonly = 0xffffffffffffffff
    msix.conf.TableOffset.Value = uint64(barno)
    msix.conf.TableOffset.Value |= uint64(msix.Pending.Size())
    msix.conf.PbaOffset.readonly = 0xffffffffffffffff
    msix.conf.PbaOffset.Value = uint64(barno)

    // Add the pci bar.
    // This includes our pending array & entries.
    pcidevice.PciBarSizes[barno] = uint32(msix.Pending.Size()) + uint32(16*vectors)
    pcidevice.PciBarOps[barno] = msix

    // Add our capability.
    // This maps to our control register.
    pcidevice.Capabilities[PciCapabilityMSIX] = &PciCapability{
        IoOperations: msix.conf,
        Size:         10,
    }

    // We're set.
    return msix
}

func (msix *MsiXDevice) Attach(vm *platform.Vm, model *Model) error {

    // Probe to find our configuration data.
    msi_conf := msix.PciDevice.Capabilities[PciCapabilityMSIX]
    msix.conf, _ = msi_conf.IoOperations.(*MsiXConf)
    if msix.conf == nil {
        // What the hell happened?
        return PciMSIError
    }

    // Reset all transient links.
    msix.conf.MsiXDevice = msix
    for _, entry := range msix.Entries {
        entry.MsiXDevice = msix
    }

    // Save our interrupt function.
    msix.msi_interrupt = func(addr platform.Paddr, data uint32) error {
        return vm.SignalMSI(addr, data, 0)
    }

    // Attach to the PciBus.
    return msix.PciDevice.Attach(vm, model)
}

func (msix *MsiXDevice) IsMSIXEnabled() bool {
    // Just check our control bit.
    // We expect callers to use this before
    // they call SendInterrupt() below.
    return msix.conf.Control.Value&0x80 != 0
}

func (msix *MsiXDevice) SendInterrupt(vector int) error {

    // Figure out our vector.
    entry := msix.FindEntry(vector)
    if entry == nil {
        // Nothing?
        msix.Debug("msix signal invalid entry?")
        return PciMSIError
    }

    if msix.IsMasked(vector) {
        // Set our pending bit.
        msix.SetPending(vector)
        return nil

    } else {
        // Clear our pending bit.
        msix.ClearPending(vector)
    }

    // Read our address and value.
    paddr := entry.Address.Value
    data := entry.Data.Value

    msix.Debug(
        "msix signal sending %x @ %x",
        entry.Data.Value,
        paddr)

    return msix.msi_interrupt(platform.Paddr(paddr), uint32(data))
}
