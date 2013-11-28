package machine

import (
    "novmm/platform"
)

//
// Virtio status.
//
const (
    VirtioStatusReboot   = 0x0
    VirtioStatusAck      = 0x1
    VirtioStatusDriver   = 0x2
    VirtioStatusDriverOk = 0x4
    VirtioStatusFailed   = 0x80
)

//
// Virtio device types.
//
const (
    VirtioTypeNet      = 1
    VirtioTypeBlock    = 2
    VirtioTypeConsole  = 3
    VirtioTypeEntropy  = 4
    VirtioTypeBalloon  = 5
    VirtioTypeIoMemory = 6
    VirtioTypeRpMsg    = 7
    VirtioTypeScsi     = 8
    VirtioType9p       = 9
)

//
// Virtio devices work by registering a device,
// and getting back a channel. This channel will
// be used to send and receive all data for the
// device -- the exact message protocol and all
// other details are completely left over for the
// device driver itself.
//
type VirtioChannels chan []byte

//
// We store the common configuration here and run
// a few different routines that allow us to multiplex
// onto the data channel. This abstracts all the H/W
// details away from the actual device logic.
//
type VirtioDevice struct {
    channels VirtioChannels

    // Our I/O base (if not PCI).
    IoBase platform.Paddr
    IoSize uint64

    // Our assigned interrupt (may be configured via PCI).
    Interrupt int
}

//
// Our configuration space constants.
//
const (
    VirtioOffsetHostCap     = 0
    VirtioOffsetGuestCap    = 4
    VirtioOffsetQueuePfn    = 8
    VirtioOffsetQueueNo     = 12
    VirtioOffsetQueueSel    = 14
    VirtioOffsetQueueNotify = 16
    VirtioOffsetStatus      = 18
    VirtioOffsetIsr         = 19
    VirtioOffsetCfgVec      = 20
    VirtioOffsetQueueVec    = 22
)

type VirtioConf struct {
    *VirtioDevice
}

func (reg *VirtioConf) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case VirtioOffsetHostCap:
    case VirtioOffsetGuestCap:
    case VirtioOffsetQueuePfn:
    case VirtioOffsetQueueNo:
    case VirtioOffsetQueueSel:
    case VirtioOffsetQueueNotify:
    case VirtioOffsetStatus:
    case VirtioOffsetIsr:
    case VirtioOffsetCfgVec:
    case VirtioOffsetQueueVec:
    }

    return 0, VirtioInvalidRegister
}

func (reg *VirtioConf) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case VirtioOffsetHostCap:
    case VirtioOffsetGuestCap:
    case VirtioOffsetQueuePfn:
    case VirtioOffsetQueueNo:
    case VirtioOffsetQueueSel:
    case VirtioOffsetQueueNotify:
    case VirtioOffsetStatus:
    case VirtioOffsetIsr:
    case VirtioOffsetCfgVec:
    case VirtioOffsetQueueVec:
    }

    return VirtioInvalidRegister
}

func NewVirtioDevice() *VirtioDevice {

    virtio := new(VirtioDevice)
    virtio.channels = make(VirtioChannels, 1024)
    return virtio
}

//
// Our vendor Id.
//
const VirtioPciVendor = 0x1af4

func LoadPciVirtioDevice(
    model *Model,
    info *DeviceInfo,
    class PciClass,
    subsystem_id uint16) (VirtioChannels, error) {

    virtio := NewVirtioDevice()
    info.Load(virtio)

    // Find our pcibus.
    var ok bool
    var pcibus *PciBus
    for _, device := range model.Devices() {
        pcibus, ok = device.info.Data.(*PciBus)
        if pcibus != nil && ok {
            break
        }
    }
    if pcibus == nil {
        return nil, VirtioPciNotFound
    }

    // Allocate our pci device.
    // NOTE: In this case, we don't actually add
    // anything to the model itself. The PciBus will
    // enumerate this device, and that's it.

    _, err := pcibus.NewDevice(
        info,
        IoMap{
            MemoryRegion{0x0, 0x100}: &VirtioConf{virtio},
        },
        PciVendorId(VirtioPciVendor),
        PciDeviceId(0x1000+subsystem_id),
        class,
        PciRevision(0),
        uint16(0),
        subsystem_id,
    )
    if err != nil {
        return nil, err
    }

    return virtio.channels, err
}

type VirtioMmioConf struct {
    class int
}

func (reg *VirtioMmioConf) Read(offset uint64, size uint) (uint64, error) {

    switch offset {
    case 0x0:
        // Magic value: 'virt'
        return 0x76697274, nil
    case 0x4:
        // Device version.
        return 1, nil
    case 0x8:
        // Device Id.
        return uint64(reg.class), nil
    case 0xc:
        // Device vendor.
        return VirtioPciVendor, nil
    }

    return 0, VirtioInvalidRegister
}

func (reg *VirtioMmioConf) Write(offset uint64, size uint, value uint64) error {
    return VirtioInvalidRegister
}

func LoadMmioVirtioDevice(
    model *Model,
    info *DeviceInfo,
    class int) (VirtioChannels, error) {

    // Create our new device.
    virtio := NewVirtioDevice()
    info.Load(virtio)

    // Find available memory.
    _, addr, err := model.Allocate(
        Reserved,
        info.Name,
        0,
        0x100,
        model.Max(),
        platform.PageSize)
    if err != nil {
        return nil, err
    }

    // Initialize static parameters.
    virtio.IoBase = addr
    virtio.IoSize = 0x100
    virtio.Interrupt = model.AllocateInterrupt()

    // Allocate our real device.
    device, err := NewDevice(
        info,
        IoMap{}, // No port-I/O.
        0,
        IoMap{
            // Carve our special ports for our device info.
            MemoryRegion{0, 0x10}: &VirtioMmioConf{class},
            // The rest will be standard virtio control registers.
            MemoryRegion{0x10, 0xe0}: &VirtioConf{virtio},
        },
        uint64(addr), // Our offset.
    )
    if err != nil {
        return nil, err
    }

    // Add it to the model.
    err = model.AddDevice(device)

    return virtio.channels, err
}
