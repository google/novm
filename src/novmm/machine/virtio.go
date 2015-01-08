// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package machine

/*
#include "virtio.h"

//
// Descriptor flags.
//
const __u16 VirtioDescFNext = VRING_DESC_F_NEXT;
const __u16 VirtioDescFWrite = VRING_DESC_F_WRITE;
const __u16 VirtioDescFIndirect = VRING_DESC_F_INDIRECT;
*/
import "C"

import (
	"encoding/json"
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
// Simple notification (for channels).
//
type VirtioNotification struct {
}

//
// The set of "in-progress" buffers.
// This is used for correctly suspending and
// resuming the active state of the device.
//
type VirtioBufferSet map[uint16]bool

//
// Virtio devices work by registering a device,
// and getting back a channel. This channel will
// be used to send and receive all data for the
// device -- the exact message protocol and all
// other details are completely left over for the
// device driver itself.
//

type VirtioChannel struct {
	*VirtioDevice `json:"-"`

	// Our channel number (set in Marshal()).
	Channel uint `json:"channel"`

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

	// Our outstanding buffers.
	Outstanding VirtioBufferSet `json:"outstanding"`

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

func (vchannel *VirtioChannel) processOne(n uint16) error {

	var buf *VirtioBuffer
	var addr C.__u64
	var length C.__u32
	var buf_flags C.__u16
	var next C.__u16
	index := C.__u16(n)

	vchannel.Debug(
		"vqueue#%d incoming slot [%d]",
		vchannel.Channel,
		index)

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
				vchannel.Channel,
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
				return err
			}

			// Append this segment.
			buf.Append(data)
		}

		// Are we finished?
		if !has_next {
			// Send these buffers.
			vchannel.Debug(
				"vqueue#%d processing slot [%d]",
				vchannel.Channel,
				buf.index)

			// Mark this as outstanding.
			vchannel.Outstanding[uint16(buf.index)] = true
			vchannel.incoming <- buf
			break

		} else {
			// Keep chaining.
			index = next
			vchannel.Debug(
				"vqueue#%d next slot [%d]",
				vchannel.Channel,
				index)
			continue
		}
	}

	// We're good.
	return nil
}

func (vchannel *VirtioChannel) consumeOne() (bool, error) {

	var flags C.__u16
	var index C.__u16
	var used_event C.__u16

	// Fetch the next buffer.
	// FIXME: We are currently not using the flags or the
	// used_event on the incoming queue. We will need to
	// support this eventually (notifying when we are short).
	if C.vring_get_buf(
		&vchannel.vring,
		C.__u16(vchannel.Consumed),
		&flags,
		&index,
		&used_event) != 0 {

		// We're up a buffer.
		vchannel.Consumed += 1

		// Process the buffer.
		err := vchannel.processOne(uint16(index))
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (vchannel *VirtioChannel) consumeOutstanding() error {

	// Resubmit outstanding buffers.
	for index, _ := range vchannel.Outstanding {
		err := vchannel.processOne(index)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vchannel *VirtioChannel) ProcessIncoming() error {

	for _ = range vchannel.notifications {
		// The device is active.
		vchannel.VirtioDevice.Acquire()

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

		// No longer active.
		vchannel.VirtioDevice.Release()
	}

	return nil
}

func (vchannel *VirtioChannel) ProcessOutgoing() error {

	for buf := range vchannel.outgoing {
		// The device is active.
		vchannel.VirtioDevice.Acquire()

		// Put in the virtqueue.
		vchannel.Debug(
			"vqueue#%d outgoing slot [%d]",
			vchannel.Channel,
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

		// Remove from our outstanding list.
		delete(vchannel.Outstanding, uint16(buf.index))

		// We can release until the next buffer comes back.
		vchannel.VirtioDevice.Release()
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
// Our channel map is just queue# => channel object.
//
type VirtioChannelMap map[uint]*VirtioChannel

//
// We store the common configuration here and run
// a few different routines that allow us to multiplex
// onto the data channel. This abstracts all the H/W
// details away from the actual device logic.
//
type VirtioDevice struct {
	Device `json:"device"`

	// Our MSI device, if valid.
	// (This is just a cast from the Device above,
	// but we save it for convenient access).
	msix *MsiXDevice

	// Our channels.
	// We expect that these will be configured
	// by the different devices.
	Channels VirtioChannelMap `json:"channels"`

	// Our configuration.
	Config *Ram `json:"config"`

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
			err := queue.QueueAddress.Write(0, size, value)
			if err != nil {
				return err
			}
			return queue.remap()
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
				err := vchannel.QueueAddress.Write(0, 8, 0)
				if err != nil {
					return err
				}
				err = vchannel.remap()
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

func (vchannel *VirtioChannel) remap() error {

	if vchannel.QueueAddress.Value != 0 {
		// Can we map this address?
		vchannel_size := C.vring_size(
			C.uint(vchannel.QueueSize.Value),
			platform.PageSize)

		mmap, err := vchannel.VirtioDevice.mmap(
			platform.Paddr(4096*vchannel.QueueAddress.Value),
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
		vchannel.Consumed = 0
	}

	return nil
}

func (vchannel *VirtioChannel) start() error {

	// Can't have size 0 or a non power of 2.
	// Ideally this wil be provided by the device.
	if vchannel.QueueSize.Value == 0 ||
		(vchannel.QueueSize.Value-1)&vchannel.QueueSize.Value != 0 {
		return VirtioInvalidQueueSize
	}

	// Ensure our mappings are correct.
	err := vchannel.remap()
	if err != nil {
		return err
	}

	// Start our goroutine which will process outgoing buffers.
	// This will add the outgoing buffers back into the vchannel.
	go vchannel.ProcessOutgoing()
	go vchannel.ProcessIncoming()

	// Is this a valid vqueue?
	// If so, then we retrigger any outstanding buffers.
	if vchannel.QueueAddress.Value != 0 {
		err := vchannel.consumeOutstanding()
		if err != nil {
			return err
		}

		// Also, we trigger a spurious notification.
		vchannel.notifications <- VirtioNotification{}
	}

	return nil
}

func (vchannel *VirtioChannel) init() {
	vchannel.incoming = make(chan *VirtioBuffer, vchannel.QueueSize.Value)
	vchannel.outgoing = make(chan *VirtioBuffer, vchannel.QueueSize.Value)
}

func NewVirtioChannel(n uint, size uint) *VirtioChannel {

	vchannel := new(VirtioChannel)
	vchannel.Channel = n
	vchannel.QueueSize.Value = uint64(size)
	vchannel.Outstanding = make(VirtioBufferSet)
	vchannel.notifications = make(chan VirtioNotification, 1)
	vchannel.init()

	return vchannel
}

func NewVirtioDevice(device Device) *VirtioDevice {
	virtio := &VirtioDevice{Device: device}
	virtio.Config = NewRam(0)
	virtio.Channels = make(VirtioChannelMap)
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

	return virtio, msix_device.init(info)
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

	return virtio, device.init(info)
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

	// See if our device is an MSI device.
	virtio.msix, _ = virtio.Device.(*MsiXDevice)

	// Ensure that all our channels are running.
	// At this point, we set the VirtioDevice pointer
	// and start up all associated goroutines for the
	// channel. We expect that NewVirtioChannel() or
	// the Marshal()/Unmarshal() routines will take
	// care of everything else.
	for _, vchannel := range virtio.Channels {
		vchannel.VirtioDevice = virtio
		err := vchannel.start()
		if err != nil {
			return err
		}
	}

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

type VirtioChannelSafe struct {
	vc *VirtioChannel
}

func (vchannel *VirtioChannelSafe) MarshalJSON() ([]byte, error) {
	return json.Marshal(vchannel.vc)
}

func (vchannel *VirtioChannelSafe) UnmarshalJSON(data []byte) error {
	vchannel.vc = NewVirtioChannel(0, 0)
	defer vchannel.vc.init()
	return json.Unmarshal(data, vchannel.vc)
}

func (chanmap *VirtioChannelMap) MarshalJSON() ([]byte, error) {

	// Create an array.
	chans := make([]VirtioChannelSafe, 0, 0)
	for _, virtio_chan := range *chanmap {
		chans = append(chans, VirtioChannelSafe{virtio_chan})
	}

	// Marshal as an array.
	return json.Marshal(chans)
}

func (chanmap *VirtioChannelMap) UnmarshalJSON(data []byte) error {

	// Unmarshal as an array.
	chans := make([]VirtioChannelSafe, 0, 0)
	err := json.Unmarshal(data, &chans)
	if err != nil {
		return err
	}

	// Load all elements.
	for _, virtio_chan := range chans {
		(*chanmap)[virtio_chan.vc.Channel] = virtio_chan.vc
	}

	return nil
}

func (set *VirtioBufferSet) MarshalJSON() ([]byte, error) {

	// Create an array.
	indices := make([]uint16, 0, len(*set))
	for index, _ := range *set {
		indices = append(indices, index)
	}

	// Marshal as an array.
	return json.Marshal(indices)
}

func (set *VirtioBufferSet) UnmarshalJSON(data []byte) error {

	// Unmarshal as an array.
	indices := make([]uint16, 0, 0)
	err := json.Unmarshal(data, &indices)
	if err != nil {
		return err
	}

	// Load all elements.
	for _, index := range indices {
		(*set)[index] = true
	}

	return nil
}
