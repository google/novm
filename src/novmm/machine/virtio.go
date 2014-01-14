package machine

/*
#include <linux/virtio_ring.h>

static inline int vring_get_buf(
    struct vring* vring,
    __u16 consumed,
    __u16* flags,
    __u16* index,
    __u16* used_event) {

    if (consumed < vring->avail->idx) {
        *flags = vring->avail->flags;
        *index = vring->avail->ring[vring->avail->idx];
        return 1;
    }

    return 0;
}

static inline void vring_get_desc(
    struct vring* vring,
    __u16 index,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next) {

    *addr = vring->desc[index].addr;
    *len = vring->desc[index].len;
    *flags = vring->desc[index].flags;
    *next = vring->desc[index].next;
}

static inline void vring_put_buf(
    struct vring* vring,
    __u16 index,
    __u32 len) {

    vring->used->ring[vring->used->idx].id = index;
    vring->used->ring[vring->used->idx].len = len;
    vring->used->idx += 1;
}

//
// Descriptor flags.
//
const __u16 VirtioDescFNext = VRING_DESC_F_NEXT;
const __u16 VirtioDescFWrite = VRING_DESC_F_WRITE;
const __u16 VirtioDescFIndirect = VRING_DESC_F_INDIRECT;
*/
import "C"

import (
    "math"
    "novmm/platform"
    "unsafe"
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
type VirtioBuffer struct {
    data  []byte
    index uint16
    write bool
}

type VirtioNotification struct {
}

type VirtioChannel struct {
    *VirtioDevice

    // Our channels.
    // Incoming is filled by the guest, and this
    // is what higher-level drivers should pay attention
    // to. Outgoing is where the buffers should be placed
    // once they are filled. The exact semantics of how
    // long buffers are held, etc. depends on the device.
    incoming      chan []VirtioBuffer
    outgoing      chan []VirtioBuffer
    notifications chan VirtioNotification

    // What index have we consumed up to?
    Consumed uint16 `json:"consumed"`

    // The queue size.
    QueueSize Register `json:"queue-size"`

    // The address written.
    QueueAddress Register `json:"queue-address"`

    // Our underlying ring.
    vring C.struct_vring
}

func (vchannel *VirtioChannel) consumePending(
    incoming chan []VirtioBuffer,
    buffer []VirtioBuffer,
    consumed uint16) ([]VirtioBuffer, uint16, error) {

    var flags C.__u16
    var index C.__u16
    var used_event C.__u16

    // Fetch all buffers.
    for C.vring_get_buf(
        &vchannel.vring,
        C.__u16(vchannel.Consumed),
        &flags,
        &index,
        &used_event) != 0 {

        // We're up a buffer.
        consumed += 1

        var addr C.__u64
        var len C.__u32
        var buf_flags C.__u16
        var next C.__u16

        for {
            // Read the entry.
            C.vring_get_desc(&vchannel.vring, index, &addr, &len, &buf_flags, &next)

            // Map the given address.
            data, err := vchannel.VirtioDevice.mmap(
                platform.Paddr(addr),
                uint64(len))
            if err != nil {
                return buffer, consumed, err
            }

            // Append our buffer.
            has_next := (buf_flags & C.VirtioDescFNext) != C.__u16(0)
            is_write := (buf_flags & C.VirtioDescFWrite) != C.__u16(0)
            buf := VirtioBuffer{data, uint16(index), is_write}
            buffer = append(buffer, buf)

            // Are we finished?
            if !has_next {
                // Send this buffer.
                incoming <- buffer
                buffer = make([]VirtioBuffer, 0, 1)

                // Interrupt the guest?
                if buf_flags == C.__u16(0) {
                    vchannel.Interrupt()
                }
                break

            } else {
                // Keep chaining.
                index = next
                continue
            }
        }
    }

    return buffer, consumed, nil
}

func (vchannel *VirtioChannel) ProcessIncoming(
    notifications chan VirtioNotification,
    incoming chan []VirtioBuffer) error {

    buffer := make([]VirtioBuffer, 0, 0)
    consumed := uint16(0)

    for _ = range notifications {
        var err error
        buffer, consumed, err = vchannel.consumePending(
            incoming,
            buffer,
            consumed)
        if err != nil {
            return err
        }
    }

    return nil
}

func (vchannel *VirtioChannel) Interrupt() error {
    return nil
}

func (vchannel *VirtioChannel) ProcessOutgoing(
    outgoing chan []VirtioBuffer) error {

    for bufs := range outgoing {
        // Put in the virtqueue.
        total_len := 0
        for _, buf := range bufs {
            total_len += len(buf.data)
        }
        C.vring_put_buf(
            &vchannel.vring,
            C.__u16(bufs[0].index),
            C.__u32(total_len))

        // Interrupt the guest.
        vchannel.Interrupt()
    }

    return nil
}

//
// We store the common configuration here and run
// a few different routines that allow us to multiplex
// onto the data channel. This abstracts all the H/W
// details away from the actual device logic.
//
type VirtioDevice struct {
    Device

    // Our channels.
    // We expect that these will be configured
    // by the different devices.
    Channels map[uint]*VirtioChannel `json:"channels"`

    // Our virtio-specific registers.
    HostFeatures  Register `json:"host-features"`
    GuestFeatures Register `json:"guest-features"`
    QueueSelect   Register `json:"queue-select"`
    QueueNotify   Register `json:"queue-notify"`
    DeviceStatus  Register `json:"device-status"`
    IsrStatus     Register `json:"isr-status"`

    // Our host map function.
    mmap func(platform.Paddr, uint64) ([]byte, error)
}

//
// Our configuration space constants.
//
const (
    VirtioOffsetHostCap     = 0
    VirtioOffsetGuestCap    = 4
    VirtioOffsetQueuePfn    = 8
    VirtioOffsetQueueSize   = 12
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
        if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
            return queue.QueueAddress.Read(0, size)
        }
        // Queue doesn't exist.
        break

    case VirtioOffsetQueueSize:
        if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
            return queue.QueueSize.Read(0, size)
        }
        // We return zero if the queue doesn't exist.
        return 0, nil

    case VirtioOffsetQueueSel:
        return reg.QueueSelect.Read(0, size)

    case VirtioOffsetQueueNotify:
        // Nothing to see here?
        break

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
        if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
            queue.SetAddress(size, value)
        }

    case VirtioOffsetQueueSize:
        // This field is read-only.
        break

    case VirtioOffsetQueueSel:
        // Simply save the selector.
        return reg.QueueSelect.Write(0, size, value)

    case VirtioOffsetQueueNotify:
        // Notify the queue if necessary.
        if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
            if queue.QueueAddress.Value != 0 {
                queue.notifications <- VirtioNotification{}
            }
        }
        return reg.QueueNotify.Write(0, size, value)

    case VirtioOffsetStatus:
        if value == VirtioStatusReboot {
            reg.Device.Debug("reboot")
        }
        if reg.DeviceStatus.Value&VirtioStatusAck == 0 &&
            value&VirtioStatusAck != 0 {
            reg.Device.Debug("ack")
        }
        if reg.DeviceStatus.Value&VirtioStatusDriver == 0 &&
            value&VirtioStatusDriver != 0 {
            reg.Device.Debug("driver")
        }
        if reg.DeviceStatus.Value&VirtioStatusDriverOk == 0 &&
            value&VirtioStatusDriverOk != 0 {
            reg.Device.Debug("driver-ok")
        }
        if reg.DeviceStatus.Value&VirtioStatusFailed == 0 &&
            value&VirtioStatusFailed != 0 {
            reg.Device.Debug("failed")
        }
        return reg.DeviceStatus.Write(0, size, value)
    case VirtioOffsetIsr:
        return reg.IsrStatus.Write(0, size, value)
    case VirtioOffsetCfgVec:
    case VirtioOffsetQueueVec:
    }

    return nil
}

func (vchannel *VirtioChannel) SetAddress(size uint, value uint64) error {

    err := vchannel.QueueAddress.Write(0, size, value)
    if err != nil {
        return err
    }

    if value != 0 {
        // Can we map this address?
        vchannel_size := C.vring_size(
            C.uint(vchannel.QueueSize.Value),
            platform.PageSize)

        mmap, err := vchannel.VirtioDevice.mmap(
            platform.Paddr(4096*value),
            uint64(vchannel_size))

        if err != nil {
            return err
        }

        // Initialize the ring.
        C.vring_init(
            &vchannel.vring,
            C.uint(vchannel.QueueSize.Value),
            unsafe.Pointer(&mmap[0]),
            platform.PageSize)

    } else {
        // Leave the address cleared. No notifcations
        // will be processed as per the Write() function.
    }

    return nil
}

func (vchannel *VirtioChannel) Init() error {
    // Can't have size 0.
    // Ideally this wil be provided by the device.
    if vchannel.QueueSize.Value == 0 {
        return VirtioInvalidQueueSize
    }

    // Recreate channels.
    vchannel.incoming = make(chan []VirtioBuffer, vchannel.QueueSize.Value)
    vchannel.outgoing = make(chan []VirtioBuffer, vchannel.QueueSize.Value)
    vchannel.notifications = make(chan VirtioNotification, 1)

    // Start our goroutine which will process outgoing buffers.
    // This will add the outgoing buffers back into the virtvchannel.
    go vchannel.ProcessOutgoing(vchannel.outgoing)
    go vchannel.ProcessIncoming(vchannel.notifications, vchannel.incoming)

    return nil
}

func (device *VirtioDevice) NewVirtioChannel(size uint) *VirtioChannel {
    vchannel := new(VirtioChannel)
    vchannel.VirtioDevice = device
    vchannel.QueueSize.Value = uint64(size)
    return vchannel
}

func NewVirtioDevice(device Device) *VirtioDevice {
    virtio := &VirtioDevice{Device: device}
    virtio.Channels = make(map[uint]*VirtioChannel)
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
        subsystem_id)
    if err != nil {
        return nil, err
    }

    virtio := NewVirtioDevice(device)
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

func (virtio *VirtioDevice) Attach(vm *platform.Vm, model *Model) error {

    // Save our map function.
    virtio.mmap = func(addr platform.Paddr, size uint64) ([]byte, error) {
        return model.Map(MemoryTypeUser, addr, size, false)
    }

    // Ensure that all our channels are reset.
    // This will do the right thing for restore.
    for _, vchannel := range virtio.Channels {
        vchannel.Init()
    }

    return virtio.Device.Attach(vm, model)
}
