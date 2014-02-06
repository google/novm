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

    // The upper address value.
    UpperAddress Register `json:"address"`

    // The table offset (& BAR).
    TableOffset Register `json:"table"`
}

type MsiXEntry struct {
    *MsiXDevice `json:"-"`

    // The lower address (+masked & pending bits).
    LowerAddress Register `json:"address"`

    // The Data.
    Data Register `json:"data"`
}

type MsiXEntries []MsiXEntry

type MsiXDevice struct {
    *PciDevice

    // This is a pointer to the pci capability.
    // NOTE: We add it and rediscover it in order
    // to avoid serialization problems.
    conf *MsiXConf

    // Our saved interrupt function.
    msi_interrupt func(addr platform.Paddr, data uint32) error

    // The entries are a device that we expose
    // to the PCI Bar as specified in the creation.
    Entries MsiXEntries `json:"entries"`
}

func (msix *MsiXEntry) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case 0:
        return msix.Data.Read(0, size)
    case 4:
        return msix.LowerAddress.Read(0, size)
    }

    return math.MaxUint64, nil
}

func (msix *MsiXEntry) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case 0:
        return msix.Data.Write(0, size, value)
    case 4:
        return msix.LowerAddress.Write(0, size, value)
    }

    return nil
}

func (msix *MsiXEntries) FindEntry(index int) *MsiXEntry {
    if index >= len(*msix) {
        return nil
    }

    return &(*msix)[index]
}

func (msix *MsiXEntries) Read(offset uint64, size uint) (uint64, error) {

    entry := msix.FindEntry(int(offset / 8))
    if entry == nil {
        return math.MaxUint64, nil
    }

    return entry.Read(offset%8, size)
}

func (msix *MsiXEntries) Write(offset uint64, size uint, value uint64) error {

    entry := msix.FindEntry(int(offset / 8))
    if entry == nil {
        return nil
    }

    return entry.Write(offset%8, size, value)
}

func (msix *MsiXConf) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case 0:
        return msix.Control.Read(0, size)
    case 2:
        return msix.UpperAddress.Read(0, size)
    case 6:
        return msix.TableOffset.Read(0, size)
    }

    return math.MaxUint64, nil
}

func (msix *MsiXConf) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case 0:
        return msix.Control.Write(0, size, value)
    case 2:
        return msix.UpperAddress.Write(0, size, value)
    case 6:
        return msix.TableOffset.Write(0, size, value)
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
        entry.LowerAddress.Value = 2      // Masked.
        entry.LowerAddress.readonly = 0x1 // Pending bit.
    }

    // Create a new set of control registers.
    msix.conf = new(MsiXConf)
    msix.conf.Control.readonly = 0xffffffffffffff7f
    msix.conf.Control.Value = uint64(vectors - 1)
    msix.conf.TableOffset.readonly = 0xffffffffffffffff
    msix.conf.TableOffset.Value = uint64(barno)

    // Add the pci bar.
    pcidevice.PciBarSizes[barno] = uint32(8 * vectors)
    pcidevice.PciBarOps[barno] = &msix.Entries

    // Add our capability.
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
    // they call SendMSIXInterrupt() below.
    return msix.conf.Control.Value&0x80 != 0
}

func (msix *MsiXDevice) SendMSIXInterrupt(data int) error {

    // Figure out our vector.
    entry := msix.Entries.FindEntry(data)
    if entry == nil {
        // Nothing?
        msix.Debug("msix signal invalid entry?")
        return PciMSIError
    }
    if entry.LowerAddress.Value&0x2 != 0 {
        // Masked.
        msix.Debug("msix signal masked")
        entry.LowerAddress.Value |= 0x1
        return nil
    }

    // Read our address.
    paddr := msix.conf.UpperAddress.Value << 32
    paddr |= entry.LowerAddress.Value & ^uint64(0x3)

    // Clear our pending bit.
    // (NOTE: I'm not sure about the exact semantics.)
    entry.LowerAddress.Value &= ^uint64(0x1)

    // Fire our interrupt.
    msix.Debug(
        "msix signal sending %x @ %x",
        entry.Data.Value,
        paddr)
    return msix.msi_interrupt(platform.Paddr(paddr), uint32(entry.Data.Value))
}
