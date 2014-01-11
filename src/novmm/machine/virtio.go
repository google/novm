package machine

import (
    "math"
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
    Device

    // Our device channels.
    channels VirtioChannels
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

    return math.MaxUint64, nil
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

    return nil
}

func NewVirtioDevice(device Device) *VirtioDevice {
    virtio := &VirtioDevice{Device: device}
    virtio.channels = make(VirtioChannels, 1024)
    return virtio
}

//
// Our vendor Id.
//
const VirtioPciVendor = 0x1af4

func NewPciVirtioDevice(
    info *DeviceInfo,
    class PciClass,
    subsystem_id uint16) (*VirtioDevice, error) {

    // Allocate our pci device.
    device, err := NewPciDevice(
        info,
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

    virtio := NewVirtioDevice(device)

    // Set our I/O region.
    device.MmioDevice.IoMap = IoMap{
        MemoryRegion{0, 0xd0}: &VirtioConf{virtio},
    }

    return virtio, device.Init(info)
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

    return math.MaxUint64, nil
}

func (reg *VirtioMmioConf) Write(offset uint64, size uint, value uint64) error {
    return nil
}

type VirtioMmioDevice struct {
    MmioDevice

    // Our assigned interrupt (may be configured via PCI).
    Interrupt int `json:"interrupt"`
}

func NewMmioVirtioDevice(
    info *DeviceInfo,
    class int) (*VirtioDevice, error) {

    // Create our Mmio device.
    device := &VirtioMmioDevice{}

    // Create our new device.
    virtio := NewVirtioDevice(device)

    // Set our I/O regions.
    device.MmioDevice.IoMap = IoMap{
        // Carve our special ports for our device info.
        MemoryRegion{0, 0x10}: &VirtioMmioConf{class},
        // The rest will be standard virtio control registers.
        MemoryRegion{0x10, 0xe0}: &VirtioConf{virtio},
    }

    return virtio, device.Init(info)
}
