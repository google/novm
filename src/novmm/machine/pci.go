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

import (
	"encoding/json"
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
type PciSubsystemVendorId PciVendorId
type PciSubsystemDeviceId PciDeviceId

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

//
// Configuration offsets.
//
const (
	PciConfigOffsetVendorId          = 0x0
	PciConfigOffsetDeviceId          = 0x2
	PciConfigOffsetCommand           = 0x4
	PciConfigOffsetStatus            = 0x6
	PciConfigOffsetRevision          = 0x8
	PciConfigOffsetProgIf            = 0x9
	PciConfigOffsetSubclassId        = 0xa
	PciConfigOffsetClassId           = 0xb
	PciConfigOffsetHeaderType        = 0xe
	PciConfigOffsetSubsystemVendorId = 0x2c
	PciConfigOffsetSubsystemDeviceId = 0x2e
	PciConfigOffsetCapabilities      = 0x34
	PciConfigOffsetInterruptLine     = 0x3c
	PciConfigOffsetInterruptPin      = 0x3d
)

//
// Our barsizes determine what we report
// for the size of each bar to the system.
//
type PciBarSizes map[uint]uint32

//
// Our bar operations will be called when the
// system maps any given bar. (The system may
// choose to map a size that is larger or smaller
// than the given bar size, this has to be handled).
//
type PciBarOps map[uint]IoOperations

//
// The capability is a generic set of IoHandlers,
// (typically a device itself or register) associated
// with some offset in the pci configuration space.
//
type PciCapability struct {

	// The pci capability id.
	Id byte `json:"id"`

	// The handlers.
	IoOperations `json:"-"`

	// The size of the data.
	Size uint64 `json:"size"`

	// The offset in configuration.
	// (NOTE: This does not include ptrs).
	Offset uint64 `json:"offset"`
}

// Our map of capabilities.
type PciCapabilityMap map[byte]*PciCapability

type PciDevice struct {
	MmioDevice

	// Packed configuration data.
	// (This encodes the vendor/device, etc.)
	Config *Ram `json:"config"`

	// Capabilities.
	// Once they have been built, we actually
	// call RefreshCapabilities to reload the map.
	Capabilities PciCapabilityMap `json:"capabilities"`

	// Bar sizes and operations.
	PciBarCount uint `json:"-"`
	PciBarSizes `json:"-"`
	PciBarOps   `json:"-"`

	// Our interrupt functions.
	std_interrupt func() error
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
	IoHandlers `json:"-"`

	// On bus devices.
	devices []*PciDevice

	// Our pci hole in memory.
	Hole MemoryRegion `json:"hole"`

	// Our configuration address.
	Addr Register `json:"config-address"`

	// Our computed offset.
	// This isn't serialized, since it will be
	// set on every run of selectLast().
	offset uint64

	// Our reserved high-mem region.
	MemoryRegion platform.Paddr `json:"reserved"`

	// The last device selected.
	// See below in PciConfAddr.Read().
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
	return reg.PciBus.Addr.Read(offset, size)
}

func (reg *PciConfAddr) Write(offset uint64, size uint, value uint64) error {
	defer reg.PciBus.selectLast()
	return reg.PciBus.Addr.Write(offset, size, value)
}

func (pcibus *PciBus) selectLast() error {

	// Load our address.
	value := pcibus.Addr.Value

	// Try to select the device.
	bus := (value >> 16) & 0x7fff
	device := (value >> 11) & 0x1f
	function := (value >> 8) & 0x7
	pcibus.offset = value & 0xff

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

	// Do we have an active device?
	if reg.PciBus.last == nil {
		return math.MaxUint64, nil
	}

	offset += (reg.PciBus.offset & 0xfc)

	// Is this a capability?
	for _, capability := range reg.PciBus.last.Capabilities {

		if offset >= capability.Offset &&
			offset < capability.Offset+capability.Size {

			value, err := capability.Read(offset-capability.Offset, size)
			reg.PciBus.last.Debug(
				"pci capabilities read [%x] %x @ %x [size: %d]",
				capability.Id,
				value,
				offset-capability.Offset,
				size)

			return value, err
		}
	}

	value, err := reg.PciBus.last.Config.Read(offset, size)

	// Debugging?
	reg.PciBus.last.Debug(
		"pci config read %x @ %x [size: %d]",
		value,
		offset,
		size)

	return value, err
}

func (reg *PciConfData) Write(offset uint64, size uint, value uint64) error {

	// Do we have an active device?
	if reg.PciBus.last == nil {
		return nil
	}

	offset += (reg.PciBus.offset & 0xfc)

	// Is this a capability?
	for _, capability := range reg.PciBus.last.Capabilities {

		if offset >= capability.Offset &&
			offset < capability.Offset+capability.Size {

			reg.PciBus.last.Debug(
				"pci capabilities write [%x] %x @ %x [size: %d]",
				capability.Id,
				value,
				offset-capability.Offset,
				size)

			return capability.Write(offset-capability.Offset, size, value)
		}
	}

	// Debugging?
	reg.PciBus.last.Debug(
		"pci config write %x @ %x [size: %d]",
		value,
		offset,
		size)

	err := reg.PciBus.last.Config.Write(offset, size, value)

	// Rebuild our BARs?
	if offset >= 0x10 &&
		offset < uint64(0x10+4*reg.PciBus.last.PciBarCount) {
		reg.PciBus.last.RebuildBars()
		return reg.PciBus.flush()
	}

	return err
}

func NewPciDevice(
	info *DeviceInfo,
	vendor_id PciVendorId,
	device_id PciDeviceId,
	class PciClass,
	revision PciRevision,
	subsystem_vendor_id PciSubsystemVendorId,
	subsystem_device_id PciSubsystemDeviceId) (*PciDevice, error) {

	// Create the pci device.
	device := new(PciDevice)
	device.Config = NewRam(0x40)
	device.PciBarSizes = make(map[uint]uint32)
	device.PciBarOps = make(map[uint]IoOperations)
	device.Capabilities = make(PciCapabilityMap)
	device.init(info)

	// Set our configuration space.
	device.Config.Set16(PciConfigOffsetVendorId, uint16(vendor_id))
	device.Config.Set16(PciConfigOffsetDeviceId, uint16(device_id))
	device.Config.Set16(PciConfigOffsetCommand, 0x143)
	device.Config.Set16(PciConfigOffsetStatus, 0x0)
	device.Config.Set8(PciConfigOffsetRevision, uint8(revision))
	device.Config.Set8(PciConfigOffsetProgIf, uint8(0))
	device.Config.Set8(PciConfigOffsetSubclassId, uint8(0))
	device.Config.Set8(PciConfigOffsetClassId, uint8(class))
	device.Config.Set8(PciConfigOffsetHeaderType, 0x0)
	device.Config.Set16(PciConfigOffsetSubsystemVendorId, uint16(subsystem_vendor_id))
	device.Config.Set16(PciConfigOffsetSubsystemDeviceId, uint16(subsystem_device_id))

	// A default device has 6 bars.
	// (This is different only for bridges, etc.)
	device.PciBarCount = 6

	// Return the pci device.
	return device, nil
}

func (pcibus *PciBus) AddDevice(device *PciDevice) error {

	// Append it to our list.
	pcibus.devices = append(pcibus.devices, device)

	// Rebuild our config-mappings.
	device.RebuildBars()

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
		MemoryRegion{0xcf8, 4}: &PciConfAddr{PciBus: bus},
		MemoryRegion{0xcfc, 4}: &PciConfData{PciBus: bus},
	}

	// Sensible default.
	// We reserve our memory hole from 256mb below 4gb to the IOApic.
	bus.Hole.Start = 0xf0000000
	bus.Hole.Size = uint64(platform.IOApic() - bus.Hole.Start)

	// Return our bus and device.
	return bus, bus.init(info)
}

func (pcibus *PciBus) Attach(vm *platform.Vm, model *Model) error {

	// This is so operating system is able to map
	// pci BARs within a 32-bit range. This is also
	// necessary because the LAPICs and IOAPICs are
	// mapped here, and it should be reserved.
	err := model.Reserve(
		vm,
		pcibus,
		MemoryTypeReserved,
		pcibus.Hole.Start,
		pcibus.Hole.Size,
		nil)
	if err != nil {
		return err
	}

	// Ensure we have a device.
	pcibus.selectLast()

	// Save the flush function.
	pcibus.flush = func() error { return model.flush() }

	return pcibus.PioDevice.Attach(vm, model)
}

func (pcidevice *PciDevice) RebuildBars() {

	// Build our IO Handlers.
	pcidevice.IoHandlers = make(IoHandlers)
	for i := uint(0); i < pcidevice.PciBarCount; i += 1 {

		barreg := int(0x10 + (i * 4))
		baraddr := pcidevice.Config.Get32(barreg)
		barsize, size_ok := pcidevice.PciBarSizes[i]
		barops, ops_ok := pcidevice.PciBarOps[i]
		if !size_ok || !ops_ok {
			// Not supported?
			pcidevice.Config.Set32(barreg, 0xffffffff)
			continue
		}

		// Mask out port-I/O bits.
		newreg := baraddr & ^(barsize-1) | 0xe

		if newreg != baraddr {
			pcidevice.Debug(
				"bar %d @ %x -> %x",
				i,
				baraddr,
				newreg)
		}

		// Rebuild our register values.
		// Save the new value.
		pcidevice.Config.Set32(barreg, newreg)

		// Create a new handler.
		region := MemoryRegion{
			platform.Paddr(baraddr & ^uint32(0xf)),
			uint64(barsize)}
		pcidevice.IoHandlers[region] = NewIoHandler(
			pcidevice,
			region.Start,
			barops)
	}
}

func (pcidevice *PciDevice) RebuildCapabilities() {

	// Already done, we don't mess with it.
	if len(pcidevice.Capabilities) > 0 &&
		pcidevice.Config.Get8(PciConfigOffsetStatus)&0x10 != 0 {
		return
	}

	// No capabilities to install.
	if len(pcidevice.Capabilities) == 0 {
		return
	}

	// Construct our pointers.
	// The end of our standard configuration is 0x40,
	// so we start our configuration pointers there.
	last_pointer := byte(0x0)
	consumed := 0x40

	for _, capability := range pcidevice.Capabilities {

		// Set this capability offset.
		pcidevice.Config.GrowTo(consumed + 2 + int(capability.Size))
		pcidevice.Config.Set8(consumed, capability.Id)
		pcidevice.Config.Set8(consumed+1, last_pointer)
		capability.Offset = uint64(consumed + 2)

		// Update our pointer.
		last_pointer = byte(consumed)
		consumed += 2 + int(capability.Size)
		if consumed%4 != 0 {
			consumed += (consumed % 4)
		}
	}

	// Save the first item,
	// and set out capabilities status bit.
	status := pcidevice.Config.Get8(PciConfigOffsetStatus)
	pcidevice.Config.Set8(PciConfigOffsetCapabilities, last_pointer)
	pcidevice.Config.Set8(PciConfigOffsetStatus, status|0x10)
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

	// Rebuild our capabilities.
	pcidevice.RebuildCapabilities()

	// FIXME: Everything uses interrupt 1.
	// This is gross, but we hard-coded the line to 1
	// unless you're using MSI. This really should be
	// fixed (if we actually plan on using PCI devices).
	pcidevice.Config.Set8(PciConfigOffsetInterruptLine, 1)
	pcidevice.Config.Set8(PciConfigOffsetInterruptPin, 1)
	pcidevice.std_interrupt = func() error {
		vm.Interrupt(platform.Irq(1), true)
		vm.Interrupt(platform.Irq(1), false)
		return nil
	}

	// Attach to the PciBus.
	return pcibus.AddDevice(pcidevice)
}

func (pcidevice *PciDevice) Interrupt() error {
	return pcidevice.std_interrupt()
}

func (capmap *PciCapabilityMap) MarshalJSON() ([]byte, error) {

	// Create an array.
	caps := make([]*PciCapability, 0, 0)
	for _, pcicap := range *capmap {
		caps = append(caps, pcicap)
	}

	// Marshal as an array.
	return json.Marshal(caps)
}

func (capmap *PciCapabilityMap) UnmarshalJSON(data []byte) error {

	// Unmarshal as an array.
	caps := make([]*PciCapability, 0, 0)
	err := json.Unmarshal(data, &caps)
	if err != nil {
		return err
	}

	// Load all elements.
	for _, pcicap := range caps {

		// Lookup the existing capability.
		usecap, ok := (*capmap)[pcicap.Id]
		if !ok {
			return PciCapabilityMismatch
		}

		// Load attributes.
		// NOTE: We don't replace the operations,
		// or the data backing them.
		usecap.Size = pcicap.Size
		usecap.Offset = pcicap.Offset
	}

	return nil
}
