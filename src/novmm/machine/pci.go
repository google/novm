package machine

import (
    "math"
    "novmm/platform"
)

//
// PciDevice --
//
// Our PCI devices are somewhat restricted compared to real
// hardware. For example, we don't support multi-function
// devices or PCI bridges. This is because our bus below is
// also supports a much more limited feature set than all PCI.

type PciVendorId uint16
type PciDeviceId uint16
type PciClass uint8
type PciRevision uint8

const (
    PciClassStorage        PciClass = 0x1
    PciClassNetwork                 = 0x2
    PciClassDisplay                 = 0x3
    PciClassMultimedia              = 0x4
    PciClassMemory                  = 0x5
    PciClassBridge                  = 0x6
    PciClassCommunications          = 0x7
    PciClassBase                    = 0x8
    PciClassInput                   = 0x9
    PciClassDocking                 = 0xa
    PciClassProcessorts             = 0xb
    PciClassSerial                  = 0xc
    PciClassMisc                    = 0xff
)

type PciDevice struct {
    MmioDevice
    InterruptDevice

    // Packed configuration data.
    // (This encodes the vendor/device, etc.)
    config NvRam

    // Bar sizes.
    bars []uint32

    // Bar operations.
    barops []IoOperations
}

//
// PciBus --
//
// Our basic PCI bus supports only a flat set of devices, and
// allows them to be single-function only. Really there's no
// practical reason to expose a hierarchy of PCI devices to the
// guest, nor to have any of them support more than one function.
// These aren't real hardware -- why should they be so complex?

type PciBus struct {
    PioDevice

    // Our Mmio Handlers.
    IoHandlers

    // On bus devices.
    devices []*PciDevice

    // The last device selected.
    // See below in PciConfAddr.Read().
    Addr   uint64 `json:"config-address"`
    Offset uint64 `json:"config-offset"`

    // The device selected.
    last *PciDevice

    // Refresh to the model flush().
    flush func() error
}

const (
    PciFunctionVendor = 0
)

type PciConfAddr struct {
    *PciBus
}
type PciConfData struct {
    *PciBus
}

func (reg *PciConfAddr) Read(offset uint64, size uint) (uint64, error) {
    return reg.PciBus.Addr, nil
}

func (reg *PciConfAddr) Write(offset uint64, size uint, value uint64) error {
    // Save the address.
    reg.PciBus.Addr = value
    return reg.PciBus.SelectLast()
}

func (pcibus *PciBus) SelectLast() error {

    // Load our address.
    value := pcibus.Addr

    // Try to select the device.
    bus := (value >> 16) & 0x7fff
    device := (value >> 11) & 0x1f
    function := (value >> 8) & 0x7
    pcibus.Offset = value & 0xff

    if bus != 0 {
        pcibus.last = nil
        return nil
    }
    if function != 0 {
        pcibus.last = nil
        return nil
    }
    if len(pcibus.devices) <= int(device) {
        pcibus.last = nil
        return nil
    }

    // Found one.
    pcibus.last = pcibus.devices[device]
    return nil
}

func (reg *PciConfData) Read(offset uint64, size uint) (uint64, error) {

    value := uint64(math.MaxUint64)

    // Do we have an active device?
    if reg.PciBus.last == nil {
        return value, nil
    }

    // Is it greater than our built-in config?
    if int(reg.PciBus.Offset) >= len(reg.PciBus.last.config) {
        // Ignore.
        return value, nil
    }

    // Is it a known register?
    switch reg.PciBus.Offset {
    }

    // Handle default.
    switch size {
    case 1:
        value = uint64(reg.PciBus.last.config.Get8(int(reg.PciBus.Offset)))
    case 2:
        value = uint64(reg.PciBus.last.config.Get16(int(reg.PciBus.Offset)))
    case 4:
        value = uint64(reg.PciBus.last.config.Get32(int(reg.PciBus.Offset)))
    }

    // Debugging?
    reg.PciBus.Debug(
        "config read %x @ %x",
        value,
        reg.PciBus.Offset)

    return value, nil
}

func (reg *PciConfData) Write(offset uint64, size uint, value uint64) error {

    // Do we have an active device?
    if reg.PciBus.last == nil {
        return nil
    }

    // Debugging?
    reg.PciBus.last.Debug(
        "config write %x @ %x",
        value,
        reg.PciBus.Offset)

    // Is it greater than our built-in config?
    if int(reg.PciBus.Offset) >= len(reg.PciBus.last.config) {
        // Ignore.
        return nil
    }

    // Handle default.
    switch size {
    case 1:
        reg.PciBus.last.config.Set8(int(reg.PciBus.Offset), uint8(value))
    case 2:
        reg.PciBus.last.config.Set16(int(reg.PciBus.Offset), uint16(value))
    case 4:
        reg.PciBus.last.config.Set32(int(reg.PciBus.Offset), uint32(value))
    }

    // Rebuild our BARs?
    if reg.PciBus.Offset >= 0x10 && reg.PciBus.Offset < 0x28 {
        reg.PciBus.last.RebuildBars()
        return reg.PciBus.flush()
    }

    return nil
}

func NewPciDevice(
    info *DeviceInfo,
    vendor_id PciVendorId,
    device_id PciDeviceId,
    class PciClass,
    revision PciRevision,
    subsystem_id uint16,
    subsystem_vendor uint16) (*PciDevice, error) {

    // Create the pci device.
    device := new(PciDevice)
    device.config = make(NvRam, 0x40, 0x40)
    device.bars = make([]uint32, 0)
    device.barops = make([]IoOperations, 0)
    device.Init(info)

    // Set our configuration space.
    device.config.Set16(0x0, uint16(vendor_id))
    device.config.Set16(0x2, uint16(device_id))
    device.config.Set8(0x8, uint8(revision))
    device.config.Set8(0x9, uint8(0)) // Prog IF.
    device.config.Set8(0xa, uint8(0)) // Subclass.
    device.config.Set8(0xb, uint8(class))
    device.config.Set8(0xe, 0x0) // Type.
    device.config.Set16(0x2c, subsystem_vendor)
    device.config.Set16(0x2e, subsystem_id)

    // Return the pci device.
    return device, nil
}

func (pcibus *PciBus) AddDevice(device *PciDevice) error {

    // Append it to our list.
    pcibus.devices = append(pcibus.devices, device)
    return pcibus.flush()
}

func (pcibus *PciBus) MmioHandlers() IoHandlers {
    return pcibus.IoHandlers
}

func NewPciBus(info *DeviceInfo) (Device, error) {

    // Create the bus.
    bus := new(PciBus)
    bus.devices = make([]*PciDevice, 0, 0)
    bus.PioDevice.IoMap = IoMap{
        // Our configuration ports.
        MemoryRegion{0xcf8, 4}: &PciConfAddr{bus},
        MemoryRegion{0xcfc, 4}: &PciConfData{bus},
    }

    // Return our bus and device.
    return bus, bus.Init(info)
}

func (pcibus *PciBus) Attach(vm *platform.Vm, model *Model) error {

    // Ensure we have a device.
    pcibus.SelectLast()

    // Save the flush function.
    pcibus.flush = func() error { return model.flush() }

    return pcibus.PioDevice.Attach(vm, model)
}

func (pcidevice *PciDevice) RebuildBars() {

    // Build our IO Handlers.
    pcidevice.IoHandlers = make(IoHandlers)
    for i := 0; i < 6; i += 1 {

        barreg := 0x10 + (i * 4)
        baraddr := pcidevice.config.Get32(barreg)
        if i >= len(pcidevice.bars) {
            // Not supported?
            pcidevice.config.Set32(barreg, 0xffffffff)
            continue
        }

        // Mask out port-I/O bits.
        newreg := baraddr & ^(pcidevice.bars[i]-1) | 0xe

        pcidevice.Debug(
            "bar %d @ %x -> %x",
            i,
            baraddr,
            newreg)

        // Rebuild our register values.
        if newreg == baraddr {
            // No change?
            continue
        }

        // Save the new value.
        pcidevice.config.Set32(barreg, newreg)

        // Create a new handler.
        region := MemoryRegion{
            platform.Paddr(baraddr & ^uint32(0xf)),
            uint64(pcidevice.bars[i])}
        pcidevice.IoHandlers[region] = NewIoHandler(
            pcidevice,
            region.Start,
            pcidevice.barops[i])
    }
}

func (pcidevice *PciDevice) Attach(vm *platform.Vm, model *Model) error {

    // Find our pcibus.
    var ok bool
    var pcibus *PciBus
    for _, device := range model.devices {
        pcibus, ok = device.(*PciBus)
        if pcibus != nil && ok {
            break
        }
    }
    if pcibus == nil {
        return PciBusNotFound
    }

    // Attach our interrupt.
    // We also set our interrupt line config appropriately.
    err := pcidevice.InterruptDevice.AttachInterrupt(vm, model)
    if err != nil {
        return err
    }

    // FIXME: Everything uses INTA.
    pcidevice.config.Set8(0x3c, 0x1)

    // uint8(pcidevice.Interrupt))

    // Attach to the PciBus.
    return pcibus.AddDevice(pcidevice)
}
