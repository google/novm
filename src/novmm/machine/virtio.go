package machine

import (
    "log"
    "math"
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
type VirtioChannel chan []byte

//
// We store the common configuration here and run
// a few different routines that allow us to multiplex
// onto the data channel. This abstracts all the H/W
// details away from the actual device logic.
//
type VirtioDevice struct {
    Device

    // Our device channels.
    // There is one set of channel
    channels []VirtioChannel

    // Our virtio-specific registers.
    HostFeatures  Register `json:"host-features"`
    GuestFeatures Register `json:"guest-features"`
    QueueAddress  Register `json:"queue-address"`
    QueueSize     Register `json:"queue-size"`
    QueueSelect   Register `json:"queue-select"`
    QueueNotify   Register `json:"queue-notify"`
    DeviceStatus  Register `json:"device-status"`
    IsrStatus     Register `json:"isr-status"`
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
        return reg.HostFeatures.Read(0, size)
    case VirtioOffsetGuestCap:
        return reg.GuestFeatures.Read(0, size)
    case VirtioOffsetQueuePfn:
        return reg.QueueAddress.Read(0, size)
    case VirtioOffsetQueueNo:
        return reg.QueueSize.Read(0, size)
    case VirtioOffsetQueueSel:
        return reg.QueueSelect.Read(0, size)
    case VirtioOffsetQueueNotify:
        return reg.QueueNotify.Read(0, size)
    case VirtioOffsetStatus:
        return reg.DeviceStatus.Read(0, size)
    case VirtioOffsetIsr:
        return reg.IsrStatus.Read(0, size)
    case VirtioOffsetCfgVec:
    case VirtioOffsetQueueVec:
    }

    return math.MaxUint64, nil
}

func (reg *VirtioConf) Write(offset uint64, size uint, value uint64) error {

    switch offset {
    case VirtioOffsetHostCap:
        return reg.HostFeatures.Write(0, size, value)
    case VirtioOffsetGuestCap:
        return reg.GuestFeatures.Write(0, size, value)
    case VirtioOffsetQueuePfn:
        return reg.QueueAddress.Write(0, size, value)
    case VirtioOffsetQueueNo:
        return reg.QueueSize.Write(0, size, value)
    case VirtioOffsetQueueSel:
        return reg.QueueSelect.Write(0, size, value)
    case VirtioOffsetQueueNotify:
        return reg.QueueNotify.Write(0, size, value)
    case VirtioOffsetStatus:
        if value == VirtioStatusReboot {
            log.Printf("%s: reboot", reg.Device.Name())
        }
        if reg.DeviceStatus.Value&VirtioStatusAck == 0 &&
            value&VirtioStatusAck != 0 {
            log.Printf("%s: ack", reg.Device.Name())
        }
        if reg.DeviceStatus.Value&VirtioStatusDriver == 0 &&
            value&VirtioStatusAck != 0 {
            log.Printf("%s: driver", reg.Device.Name())
        }
        if reg.DeviceStatus.Value&VirtioStatusDriverOk == 0 &&
            value&VirtioStatusAck != 0 {
            log.Printf("%s: driver-ack", reg.Device.Name())
        }
        if reg.DeviceStatus.Value&VirtioStatusFailed == 0 &&
            value&VirtioStatusAck != 0 {
            log.Printf("%s: failed", reg.Device.Name())
        }
        return reg.DeviceStatus.Write(0, size, value)
    case VirtioOffsetIsr:
        return reg.IsrStatus.Write(0, size, value)
    case VirtioOffsetCfgVec:
    case VirtioOffsetQueueVec:
    }

    return nil
}

func NewVirtioDevice(device Device, channels []uint) *VirtioDevice {
    virtio := &VirtioDevice{Device: device}
    virtio.channels = make([]VirtioChannel, len(channels), len(channels))
    for i := 0; i < len(channels); i += 1 {
        virtio.channels[i] = make(VirtioChannel, channels[i])
    }
    return virtio
}

//
// Our vendor Id.
//
const VirtioPciVendor = 0x1af4

func NewPciVirtioDevice(
    info *DeviceInfo,
    channels []uint,
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

    virtio := NewVirtioDevice(device, channels)
    device.bars = []uint32{uint32(platform.PageSize)}
    device.barops = []IoOperations{&VirtioConf{virtio}}

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
    channels []uint,
    class int) (*VirtioDevice, error) {

    // Create our Mmio device.
    device := &VirtioMmioDevice{}

    // Create our new device.
    virtio := NewVirtioDevice(device, channels)

    // Set our I/O regions.
    device.MmioDevice.IoMap = IoMap{
        // Carve our special ports for our device info.
        MemoryRegion{0, 0x10}: &VirtioMmioConf{class},
        // The rest will be standard virtio control registers.
        MemoryRegion{0x10, 0xe0}: &VirtioConf{virtio},
    }

    return virtio, device.Init(info)
}
