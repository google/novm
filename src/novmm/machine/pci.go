package machine

import (
    "log"
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
    // Generic info.
    info *DeviceInfo

    // Packed configuration data.
    // (This encodes the vendor/device, etc.)
    conf []byte

    // The config handlers.
    PciConfig IoHandlers

    // Our device lookup cache.
    // (Used for accessing device conf data.)
    cache *IoCache
}

func (pcidev *PciDevice) SetConf8(offset int, data uint8) {
    pcidev.conf[offset] = byte(data)
}
func (pcidev *PciDevice) GetConf8(offset int) uint8 {
    return pcidev.conf[offset]
}

func (pcidev *PciDevice) SetConf16(offset int, data uint16) {
    pcidev.conf[offset] = byte(data & 0xff)
    pcidev.conf[offset+1] = byte((data >> 8) & 0xff)
}
func (pcidev *PciDevice) GetConf16(offset int) uint16 {
    return (uint16(pcidev.conf[offset]) |
        (uint16(pcidev.conf[offset+1]) << 8))
}

func (pcidev *PciDevice) SetConf32(offset int, data uint32) {
    pcidev.conf[offset] = byte(data & 0xff)
    pcidev.conf[offset+1] = byte((data >> 8) & 0xff)
    pcidev.conf[offset+2] = byte((data >> 16) & 0xff)
    pcidev.conf[offset+3] = byte((data >> 24) & 0xff)
}
func (pcidev *PciDevice) GetConf32(offset int) uint32 {
    return (uint32(pcidev.conf[offset]) |
        (uint32(pcidev.conf[offset+1]) << 8) |
        (uint32(pcidev.conf[offset+2]) << 16) |
        (uint32(pcidev.conf[offset+3]) << 24))
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
    // On bus devices.
    devices []*PciDevice

    // The last device selected.
    // See below in PciConfAddr.Read().
    addr   uint64     // The actual address.
    offset uint64     // The offset component.
    last   *PciDevice // The device selected.
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
    return reg.PciBus.addr, nil
}

func (reg *PciConfAddr) Write(offset uint64, size uint, value uint64) error {
    // Save the address.
    reg.PciBus.addr = value

    // Try to select the device.
    bus := (value >> 16) & 0x7fff
    device := (value >> 11) & 0x1f
    function := (value >> 8) & 0x7
    reg.PciBus.offset = value & 0xff

    if bus != 0 {
        reg.PciBus.last = nil
        return nil
    }
    if function != 0 {
        reg.PciBus.last = nil
        return nil
    }
    if len(reg.PciBus.devices) <= int(device) {
        reg.PciBus.last = nil
        return nil
    }

    // Found one.
    reg.PciBus.last = reg.PciBus.devices[device]
    return nil
}

type PciEvent struct {
    size  uint
    data  uint64
    write bool
}

func (mmio PciEvent) Size() uint {
    return mmio.size
}

func (mmio PciEvent) GetData() uint64 {
    return mmio.data
}

func (mmio PciEvent) SetData(val uint64) {
    mmio.data = val
}

func (mmio PciEvent) IsWrite() bool {
    return mmio.write
}

func (reg *PciConfData) Read(offset uint64, size uint) (uint64, error) {

    value := uint64(math.MaxUint64)

    // Do we have an active device?
    if reg.PciBus.last == nil {
        return value, nil
    }

    // Is it greater than our built-in config?
    if int(reg.PciBus.offset) >= len(reg.PciBus.last.conf) {
        // Submit to the device handler.
        addr := platform.Paddr(int(reg.PciBus.offset) - len(reg.PciBus.last.conf))
        handler := reg.PciBus.last.cache.lookup(addr)
        io_event := &PciEvent{size, 0, false}
        err := handler.queue.Submit(
            io_event,
            addr.OffsetFrom(handler.MemoryRegion.Start))
        if err != nil {
            return value, err
        }
        value = io_event.data

    } else {
        // Is it a known register?
        switch reg.PciBus.offset {
        }

        // Handle default.
        switch size {
        case 1:
            value = uint64(reg.PciBus.last.GetConf8(int(reg.PciBus.offset)))
        case 2:
            value = uint64(reg.PciBus.last.GetConf16(int(reg.PciBus.offset)))
        case 4:
            value = uint64(reg.PciBus.last.GetConf32(int(reg.PciBus.offset)))
        }
    }

    // Debugging?
    if reg.PciBus.last.info.Debug {
        log.Printf("pci-bus:%s: config read %x @ %x",
            reg.PciBus.last.info.Name,
            value,
            reg.PciBus.offset)
    }

    return value, nil
}

func (reg *PciConfData) Write(offset uint64, size uint, value uint64) error {

    // Do we have an active device?
    if reg.PciBus.last == nil {
        return nil
    }

    // Debugging?
    if reg.PciBus.last.info.Debug {
        log.Printf("pci-bus:%s: config write %x @ %x",
            reg.PciBus.last.info.Name,
            value,
            offset)
    }

    // Is it greater than our built-in config?
    if int(reg.PciBus.offset) >= len(reg.PciBus.last.conf) {
        // Submit to the device handler.
        addr := platform.Paddr(int(reg.PciBus.offset) - len(reg.PciBus.last.conf))
        handler := reg.PciBus.last.cache.lookup(addr)
        io_event := &PciEvent{size, value, true}
        return handler.queue.Submit(
            io_event,
            addr.OffsetFrom(handler.MemoryRegion.Start))
    }

    // Is it a known register?
    switch reg.PciBus.offset {
    }

    // Handle default.
    switch size {
    case 1:
        reg.PciBus.last.SetConf8(int(reg.PciBus.offset), uint8(value))
    case 2:
        reg.PciBus.last.SetConf16(int(reg.PciBus.offset), uint16(value))
    case 4:
        reg.PciBus.last.SetConf32(int(reg.PciBus.offset), uint32(value))
    }

    return nil
}

func (bus *PciBus) NewDevice(
    info *DeviceInfo,
    config IoMap,
    vendor_id PciVendorId,
    device_id PciDeviceId,
    class PciClass,
    revision PciRevision,
    subsystem_id uint16,
    subsystem_vendor uint16) (*PciDevice, error) {

    // Create the pci device.
    pci_device := new(PciDevice)
    pci_device.info = info
    pci_device.PciConfig = make(IoHandlers)
    pci_device.conf = make([]byte, 0x40, 0x40)
    for region, ops := range config {
        pci_device.PciConfig[region] = NewIoHandler(
            pci_device.info, region, ops)
    }
    pci_device.cache = NewIoCache([]*IoHandlers{&pci_device.PciConfig})

    // Set our configuration space.
    pci_device.SetConf16(0x0, uint16(vendor_id))
    pci_device.SetConf16(0x2, uint16(device_id))
    pci_device.SetConf8(0x8, uint8(revision))
    pci_device.SetConf8(0x9, uint8(0)) // Prog IF.
    pci_device.SetConf8(0xa, uint8(0)) // Subclass.
    pci_device.SetConf8(0xb, uint8(class))

    pci_device.SetConf8(0xe, 0x0) // Type.
    pci_device.SetConf16(0x2c, subsystem_vendor)
    pci_device.SetConf16(0x2e, subsystem_id)

    // Append it to our list.
    bus.devices = append(bus.devices, pci_device)

    // Return the pci device.
    return pci_device, nil
}

func NewPciBus(info *DeviceInfo) (*PciBus, *Device, error) {

    // Create the bus.
    bus := new(PciBus)
    bus.devices = make([]*PciDevice, 0, 0)
    info.Load(bus)

    // Create a bus device.
    hostbridge, err := bus.NewDevice(
        info,
        IoMap{},
        PciVendorId(0x1022), // AMD.
        PciDeviceId(0x7432), // Made-up.
        PciClassBridge,
        PciRevision(0),
        0,
        0)
    if err != nil {
        return nil, nil, err
    }
    hostbridge.conf[0x6] = 0x10 // Caps present.
    hostbridge.conf[0xe] = 1    // Type.

    // Add our capabilities.
    hostbridge.conf[0x34] = 0x40                          // Cap pointer.
    hostbridge.conf = append(hostbridge.conf, byte(0x40)) // Type port root.
    hostbridge.conf = append(hostbridge.conf, byte(0))    // End of cap pointer.

    // Create the device.
    device, err := NewDevice(
        info,
        IoMap{
            // Our configuration ports.
            MemoryRegion{0xcf8, 4}: &PciConfAddr{bus},
            MemoryRegion{0xcfc, 4}: &PciConfData{bus},
        },
        0,  // Port-I/O offset.
        IoMap{},
        0,  // Memory-I/O offset.
    )

    // Return our bus and device.
    return bus, device, err
}

func LoadPciBus(model *Model, info *DeviceInfo) error {

    _, device, err := NewPciBus(info)
    if err != nil {
        return err
    }

    return model.AddDevice(device)
}
