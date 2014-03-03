package machine

/*
#include <linux/virtio_ring.h>

static inline int vring_get_buf(
    struct vring* vring,
    __u16 consumed,
    __u16* flags,
    __u16* index,
    __u16* used_event) {

    if (consumed != vring->avail->idx) {

        if (consumed+1 < vring->avail->idx) {
            vring->used->flags = VRING_USED_F_NO_NOTIFY;
        } else {
            vring->used->flags = 0;
            vring_avail_event(vring) = consumed+1;
        }

        *flags = vring->avail->flags;
        *index = vring->avail->ring[consumed%vring->num];

        return 1;
    }

    return 0;
}

static inline void vring_read_desc(
    struct vring_desc* desc,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next) {

    *addr = desc->addr;
    *len = desc->len;
    *flags = desc->flags;
    *next = desc->next;
}

static inline void vring_get_index(
    struct vring* vring,
    __u16 index,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next) {

    vring_read_desc(&vring->desc[index], addr, len, flags, next);
}

static inline void vring_put_buf(
    struct vring* vring,
    __u16 index,
    __u32 len,
    int* evt_interrupt,
    int* no_interrupt) {

    vring->used->ring[vring->used->idx%vring->num].id = index;
    vring->used->ring[vring->used->idx%vring->num].len = len;
    *evt_interrupt = vring_used_event(vring) == vring->used->idx;
    *no_interrupt = vring->used->flags & VRING_AVAIL_F_NO_INTERRUPT;

    asm volatile ("" : : : "memory");
    vring->used->idx += 1;
    asm volatile ("" : : : "memory");
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
    "log"
    "math"
    "novmm/platform"
    "sync/atomic"
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
// Generic features.
//
const (
    VirtioRingFEventIdx = 1 << 29
)

//
// Virtio devices work by registering a device,
// and getting back a channel. This channel will
// be used to send and receive all data for the
// device -- the exact message protocol and all
// other details are completely left over for the
// device driver itself.
//

type VirtioNotification struct {
}

type VirtioChannel struct {
    *VirtioDevice

    // Our channel number (set in Init()).
    channelno uint

    // Our channels.
    // Incoming is filled by the guest, and this
    // is what higher-level drivers should pay attention
    // to. Outgoing is where the buffers should be placed
    // once they are filled. The exact semantics of how
    // long buffers are held, etc. depends on the device.
    incoming chan *VirtioBuffer
    outgoing chan *VirtioBuffer

    // Notification channel.
    // This is used internally for notifications that
    // drive us consuming buffers. It is not intended
    // for any device-specific code to use.
    notifications chan VirtioNotification

    // Currently pending notifications?
    pending int32

    // What index have we consumed up to?
    Consumed uint16 `json:"consumed"`

    // The queue size.
    QueueSize Register `json:"queue-size"`

    // The address written.
    QueueAddress Register `json:"queue-address"`

    // Our MSI-X Vectors.
    CfgVec   Register `json:"config-vector"`
    QueueVec Register `json:"queue-vector"`

    // Our underlying ring.
    vring C.struct_vring
}

func (vchannel *VirtioChannel) consumeOne() (bool, error) {

    var flags C.__u16
    var index C.__u16
    var used_event C.__u16

    // Fetch the next buffer.
    if C.vring_get_buf(
        &vchannel.vring,
        C.__u16(vchannel.Consumed),
        &flags,
        &index,
        &used_event) != 0 {

        // We're up a buffer.
        vchannel.Debug(
            "vqueue#%d incoming slot [%d]",
            vchannel.channelno,
            index)
        vchannel.Consumed += 1

        var buf *VirtioBuffer
        var addr C.__u64
        var length C.__u32
        var buf_flags C.__u16
        var next C.__u16

        for {
            // Read the entry.
            C.vring_get_index(
                &vchannel.vring,
                index,
                &addr,
                &length,
                &buf_flags,
                &next)

            // Append our buffer.
            has_next := (buf_flags & C.__u16(C.VirtioDescFNext)) != C.__u16(0)
            is_write := (buf_flags & C.__u16(C.VirtioDescFWrite)) != C.__u16(0)
            is_indirect := (buf_flags & C.__u16(C.VirtioDescFIndirect)) != C.__u16(0)

            // Do we have a buffer?
            if buf == nil {
                buf = NewVirtioBuffer(uint16(index), !is_write)
            }

            if is_indirect {
                // FIXME: Map all indirect buffers.
                log.Printf("WARNING: Indirect buffers not supported.")

            } else {
                // Map the given address.
                vchannel.Debug("vqueue#%d map [%x-%x]",
                    vchannel.channelno,
                    platform.Paddr(addr),
                    uint64(addr)+uint64(length)-1)

                data, err := vchannel.VirtioDevice.mmap(
                    platform.Paddr(addr),
                    uint64(length))

                if err != nil {
                    log.Printf(
                        "Unable to map [%x,%x]? Flags are %x, next is %x.",
                        addr,
                        addr+C.__u64(length)-1,
                        buf_flags,
                        next)
                    return false, err
                }

                // Append this segment.
                buf.Append(data)
            }

            // Are we finished?
            if !has_next {
                // Send these buffers.
                vchannel.Debug(
                    "vqueue#%d processing slot [%d]",
                    vchannel.channelno,
                    buf.index)

                vchannel.incoming <- buf
                break

            } else {
                // Keep chaining.
                index = next
                vchannel.Debug(
                    "vqueue#%d next slot [%d]",
                    vchannel.channelno,
                    index)
                continue
            }
        }

        return true, nil
    }

    return false, nil
}

func (vchannel *VirtioChannel) ProcessIncoming() error {

    for _ = range vchannel.notifications {

        // Reset our pending variable.
        // A write to the notification register
        // will drop a notification in the channel
        // again -- which is okay as the channel
        // should now be empty.
        atomic.StoreInt32(&vchannel.pending, 0)

        for {
            found, err := vchannel.consumeOne()
            if err != nil {
                return err
            }
            if !found {
                break
            }
        }
    }

    return nil
}

func (vchannel *VirtioChannel) ProcessOutgoing() error {

    for buf := range vchannel.outgoing {

        // Put in the virtqueue.
        vchannel.Debug(
            "vqueue#%d outgoing slot [%d]",
            vchannel.channelno,
            buf.index)

        var evt_interrupt C.int
        var no_interrupt C.int
        C.vring_put_buf(
            &vchannel.vring,
            C.__u16(buf.index),
            C.__u32(buf.length),
            &evt_interrupt,
            &no_interrupt)

        if vchannel.HasFeatures(VirtioRingFEventIdx) {
            // This is used the event index.
            if evt_interrupt != C.int(0) {
                // Interrupt the guest.
                vchannel.Interrupt(true)
            }
        } else {
            // We have no event index.
            if no_interrupt == C.int(0) {
                // Interrupt the guest.
                vchannel.Interrupt(true)
            }
        }
    }

    return nil
}

func (vchannel *VirtioChannel) Interrupt(queue bool) {
    if vchannel.VirtioDevice.IsMSIXEnabled() {
        if queue {
            // Send on the specified queue vector.
            if vchannel.QueueVec.Value != 0xffff {
                vchannel.VirtioDevice.msix.SendInterrupt(int(vchannel.QueueVec.Value))
            }
        } else {
            // Send on the specified config vector.
            if vchannel.CfgVec.Value != 0xffff {
                vchannel.VirtioDevice.msix.SendInterrupt(int(vchannel.CfgVec.Value))
            }
        }
    } else {
        // Just send a standard interrupt.
        vchannel.VirtioDevice.Interrupt()
    }
}

//
// We store the common configuration here and run
// a few different routines that allow us to multiplex
// onto the data channel. This abstracts all the H/W
// details away from the actual device logic.
//
type VirtioDevice struct {
    Device

    // Our MSI device, if valid.
    msix *MsiXDevice

    // Our channels.
    // We expect that these will be configured
    // by the different devices.
    Channels map[uint]*VirtioChannel `json:"channels"`

    // Our configuration.
    Config Ram `json:"config"`

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
    VirtioOffsetHostCap     = 0x00
    VirtioOffsetGuestCap    = 0x04
    VirtioOffsetQueuePfn    = 0x08
    VirtioOffsetQueueSize   = 0x0c
    VirtioOffsetQueueSel    = 0x0e
    VirtioOffsetQueueNotify = 0x10
    VirtioOffsetStatus      = 0x12
    VirtioOffsetIsr         = 0x13
    VirtioOffsetCfgVec      = 0x14
    VirtioOffsetQueueVec    = 0x16
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

    default:
        if reg.VirtioDevice.IsMSIXEnabled() {
            switch offset {
            case VirtioOffsetCfgVec:
                if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
                    return queue.CfgVec.Read(0, size)
                }
                return math.MaxUint64, nil
            case VirtioOffsetQueueVec:
                if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
                    return queue.QueueVec.Read(0, size)
                }
                return math.MaxUint64, nil
            default:
                reg.Debug(
                    "virtio read @ %x->%x (msi-enabled)",
                    offset,
                    offset-VirtioOffsetQueueVec-2)
                return reg.VirtioDevice.Config.Read(
                    offset-VirtioOffsetQueueVec-2,
                    size)
            }
        } else {
            reg.Debug(
                "virtio read @ %x->%x (no-msi-enabled)",
                offset,
                offset-VirtioOffsetIsr-1)
            return reg.VirtioDevice.Config.Read(
                offset-VirtioOffsetIsr-1,
                size)
        }
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
            return queue.SetAddress(size, value)
        }

    case VirtioOffsetQueueSize:
        // This field is read-only.
        break

    case VirtioOffsetQueueSel:
        // Simply save the selector.
        return reg.QueueSelect.Write(0, size, value)

    case VirtioOffsetQueueNotify:
        // Notify the queue if necessary.
        if queue, ok := reg.VirtioDevice.Channels[uint(value)]; ok {
            if queue.QueueAddress.Value != 0 {
                // Do we need a notification?
                // We do this to avoid blocking when there are
                // already pending notifications in the channel.
                if atomic.CompareAndSwapInt32(&queue.pending, 0, 1) {
                    queue.notifications <- VirtioNotification{}
                }
            }
        }
        err := reg.QueueNotify.Write(0, size, value)
        if err != nil {
            return err
        }

        // This is a saveable register.
        return SaveIO

    case VirtioOffsetStatus:
        if value == VirtioStatusReboot {
            reg.Device.Debug("reboot")
            for _, vchannel := range reg.VirtioDevice.Channels {
                err := vchannel.SetAddress(4, 0)
                if err != nil {
                    return err
                }
            }
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

    default:
        if reg.VirtioDevice.IsMSIXEnabled() {
            switch offset {
            case VirtioOffsetCfgVec:
                if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
                    return queue.CfgVec.Write(0, size, value)
                }
                return nil
            case VirtioOffsetQueueVec:
                if queue, ok := reg.VirtioDevice.Channels[uint(reg.QueueSelect.Value)]; ok {
                    return queue.QueueVec.Write(0, size, value)
                }
                return nil
            default:
                reg.Debug(
                    "virtio write @ %x->%x (msi-enabled)",
                    offset,
                    offset-VirtioOffsetQueueVec-2)
                return reg.VirtioDevice.Config.Write(
                    offset-VirtioOffsetQueueVec-2,
                    size,
                    value)
            }
        } else {
            reg.Debug(
                "virtio write @ %x->%x (no-msi-enabled)",
                offset,
                offset-VirtioOffsetIsr-1)
            return reg.VirtioDevice.Config.Write(
                offset-VirtioOffsetIsr-1,
                size,
                value)
        }
    }

    return nil
}

func (vchannel *VirtioChannel) SetAddress(
    size uint,
    value uint64) error {

    err := vchannel.QueueAddress.Write(0, size, value)
    if err != nil {
        return err
    }

    // Reset our consumed amount.
    vchannel.Consumed = 0

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

        // Notify the consumer.
        vchannel.notifications <- VirtioNotification{}

    } else {
        // Leave the address cleared. No notifcations
        // will be processed as per the Write() function.
    }

    return nil
}

func (vchannel *VirtioChannel) Init(n uint) error {

    // Save our channel number.
    vchannel.channelno = n

    // Can't have size 0 or a non power of 2.
    // Ideally this wil be provided by the device.
    if vchannel.QueueSize.Value == 0 ||
        (vchannel.QueueSize.Value-1)&vchannel.QueueSize.Value != 0 {
        return VirtioInvalidQueueSize
    }

    // Recreate channels.
    vchannel.incoming = make(chan *VirtioBuffer)
    vchannel.outgoing = make(chan *VirtioBuffer)
    vchannel.notifications = make(chan VirtioNotification, 1)

    // Start our goroutine which will process outgoing buffers.
    // This will add the outgoing buffers back into the virtvchannel.
    go vchannel.ProcessOutgoing()
    go vchannel.ProcessIncoming()

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
    virtio.Config = make(Ram, 0, 0)
    virtio.Channels = make(map[uint]*VirtioChannel)
    virtio.IsrStatus.readclr = 0x1
    virtio.SetFeatures(VirtioRingFEventIdx)
    return virtio
}

//
// Our vendor Id.
//
const VirtioPciVendor = 0x1af4

func NewPciVirtioDevice(
    info *DeviceInfo,
    class PciClass,
    subsystem_id PciSubsystemDeviceId,
    vectors uint) (*VirtioDevice, error) {

    // Allocate our pci device.
    device, err := NewPciDevice(
        info,
        PciVendorId(VirtioPciVendor),
        PciDeviceId(0x1000+subsystem_id),
        class,
        PciRevision(0),
        PciSubsystemVendorId(0),
        subsystem_id)
    if err != nil {
        return nil, err
    }

    // Create an MSI-enabled device.
    msix_device := NewMsiXDevice(device, 5, vectors)
    virtio := NewVirtioDevice(msix_device)
    device.PciBarSizes[0] = platform.PageSize
    device.PciBarOps[0] = &VirtioConf{virtio}

    return virtio, msix_device.Init(info)
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

func (virtio *VirtioDevice) SetFeatures(features uint32) {
    virtio.HostFeatures.Value = virtio.HostFeatures.Value | uint64(features)
}

func (virtio *VirtioDevice) HasFeatures(features uint32) bool {
    return (virtio.GetFeatures() & features) == features
}

func (virtio *VirtioDevice) GetFeatures() uint32 {
    return uint32(virtio.GuestFeatures.Value & virtio.HostFeatures.Value)
}

func (virtio *VirtioDevice) Attach(vm *platform.Vm, model *Model) error {

    // Save our map function.
    virtio.mmap = func(addr platform.Paddr, size uint64) ([]byte, error) {
        return model.Map(MemoryTypeUser, addr, size, false)
    }

    // Ensure that all our channels are reset.
    // This will do the right thing for restore.
    for n, vchannel := range virtio.Channels {
        err := vchannel.Init(n)
        if err != nil {
            return err
        }
    }

    // See if our device is an MSI device.
    virtio.msix, _ = virtio.Device.(*MsiXDevice)

    return virtio.Device.Attach(vm, model)
}

func (virtio *VirtioDevice) IsMSIXEnabled() bool {
    return virtio.msix != nil && virtio.msix.IsMSIXEnabled()
}

func (virtio *VirtioDevice) Interrupt() error {
    // Just send a standrd interrupt,
    // with an updated status register.
    virtio.IsrStatus.Value = virtio.IsrStatus.Value | 0x1
    return virtio.Device.Interrupt()
}
