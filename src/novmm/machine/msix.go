package machine

import (
    "math"
    "novmm/platform"
)

const (
    PciCapabilityMSIX = 0x11
)

const (
    PciMsiXControlMasked = 0x4000
    PciMsiXControlEnable = 0x8000
)

const (
    PciMsiXEntryControlMasked = 0x01
)

const (
    PciMsiXEntrySize = 0x10
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
    case 0x0:
        fallthrough
    case 0x4:
        value, err := msix.Address.Read(offset, size)
        msix.Debug("msix-entry read address %x @ %x", value, offset)
        return value, err
    case 0x8:
        value, err := msix.Data.Read(0, size)
        msix.Debug("msix-entry read data %x @ %x", value, offset)
        return value, err
    case 0xc:
        value, err := msix.Control.Read(0, size)
        msix.Debug("msix-entry read control %x @ %x", value, offset)
        return value, err
    default:
        msix.Debug("msix-entry read invalid @ %x?", offset)
    }

    return math.MaxUint64, nil
}

func (msix *MsiXEntry) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case 0x0:
        fallthrough
    case 0x4:
        msix.Debug("msix-entry write address %x @ %x", value, offset)
        return msix.Address.Write(offset, size, value)
    case 0x8:
        msix.Debug("msix-entry write data %x @ %x", value, offset)
        return msix.Data.Write(0, size, value)
    case 0xc:
        msix.Debug("msix-entry write control %x @ %x", value, offset)
        return msix.Control.Write(0, size, value)
    default:
        msix.Debug("msix-entry write invalid %x @ %x?", value, offset)
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
    if msix.conf.Control.Value&PciMsiXControlMasked != 0 {
        return true
    }
    entry := msix.FindEntry(int(vector / 8))
    if entry != nil && entry.Control.Value&PciMsiXEntryControlMasked != 0 {
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

    // Is this a pending bit?
    if offset < uint64(msix.Pending.Size()) {
        msix.Debug("msix read pending bit @ %x", offset)
        return msix.Pending.Read(offset, size)
    }
    offset -= uint64(msix.Pending.Size())

    // Is it an entry?
    entry := msix.FindEntry(int(offset / PciMsiXEntrySize))
    if entry == nil {
        return math.MaxUint64, nil
    }

    // FIXME:
    // Why is this being reset?
    entry.MsiXDevice = msix

    return entry.Read(offset%PciMsiXEntrySize, size)
}

func (msix *MsiXDevice) Write(offset uint64, size uint, value uint64) error {

    // Is this a pending bit?
    if offset < uint64(msix.Pending.Size()) {
        msix.Debug("msix write pending bit @ %x", offset)
        return msix.Pending.Write(offset, size, value)
    }
    offset -= uint64(msix.Pending.Size())

    // Is this an entry?
    entry := msix.FindEntry(int(offset / PciMsiXEntrySize))
    if entry == nil {
        return nil
    }

    // FIXME:
    // Why is this being reset?
    entry.MsiXDevice = msix

    defer msix.CheckPending(int(offset / PciMsiXEntrySize))
    return entry.Write(offset%PciMsiXEntrySize, size, value)
}

func (msix *MsiXConf) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case 0:
        value, err := msix.Control.Read(0, size)
        msix.Debug("msix read control %x @ %x", value, offset)
        return value, err
    case 2:
        value, err := msix.TableOffset.Read(0, size)
        msix.Debug("msix read table-offset %x @ %x", value, offset)
        return value, err
    case 6:
        value, err := msix.PbaOffset.Read(0, size)
        msix.Debug("msix read pba-offset %x @ %x", value, offset)
        return value, err
    }

    return math.MaxUint64, nil
}

func (msix *MsiXConf) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case 0:
        msix.Debug("msix write control %x @ %x", value, offset)
        defer msix.MsiXDevice.CheckAllPending()
        return msix.Control.Write(0, size, value)
    case 2:
        msix.Debug("msix write table-offset %x @ %x", value, offset)
        return msix.TableOffset.Write(0, size, value)
    case 6:
        msix.Debug("msix write pba-offset %x @ %x", value, offset)
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
        entry.MsiXDevice = msix
        entry.Control.Value = PciMsiXControlMasked
    }

    // Create our new pending bit array.
    // NOTE: This doesn't have any fixed size, as it
    // is simply a large bitmask. However, we ensure that
    // it aligns on a 64-bit boundary, to make accesses to
    // the entries that follow the pending bits convenient.
    pending_size := 16 * int((vectors+63)/64)
    msix.Pending = make(Ram, 0, 0)
    msix.Pending.GrowTo(pending_size)

    // Create a new set of control registers.
    msix.conf = new(MsiXConf)
    msix.conf.Control.readonly = math.MaxUint64 & ^PciMsiXControlEnable
    msix.conf.Control.Value = uint64(vectors - 1)
    msix.conf.TableOffset.readonly = math.MaxUint64
    msix.conf.TableOffset.Value = uint64(barno)
    msix.conf.TableOffset.Value |= uint64(msix.Pending.Size())
    msix.conf.PbaOffset.readonly = math.MaxUint64
    msix.conf.PbaOffset.Value = uint64(barno)

    // Add the pci bar.
    // This includes our pending array & entries.
    bar_size := uint32(msix.Pending.Size()) + uint32(PciMsiXEntrySize*vectors)
    if bar_size < platform.PageSize {
        bar_size = platform.PageSize
    }
    pcidevice.PciBarOps[barno] = msix
    pcidevice.PciBarSizes[barno] = bar_size

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
    return msix.conf.Control.Value&PciMsiXControlEnable != 0
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
