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
	"crypto/rand"
	"net"
	"novmm/platform"
)

//
// Virtio Net Features
//
const (
	VirtioNetFCsum     uint32 = 1 << 0
	VirtioNetFMac             = 1 << 5
	VirtioNetFHostTso4        = 1 << 11
	VirtioNetFHostTso6        = 1 << 12
	VirtioNetFHostEcn         = 1 << 13
	VirtioNetFHostUfo         = 1 << 14
	VirtioNetFStatus          = 1 << 16
)

//
// VirtioNet Status Bits
//
const (
	VirtioNetLinkUp   = (1 << 0)
	VirtioNetAnnounce = (1 << 1)
)

//
// VirtioNet Config Space
//
const (
	VirtioNetMacOffset    = 0
	VirtioNetMacLen       = 6
	VirtioNetStatusOffset = (VirtioNetMacOffset + VirtioNetMacLen)
	VirtioNetStatusLen    = 2
	VirtioNetConfigLen    = (VirtioNetMacLen + VirtioNetStatusLen)
)

//
// VirtioNet VLAN support
//
const (
	VirtioNetHeaderSize = 10
)

type VirtioNetDevice struct {
	*VirtioDevice

	// The tap device file descriptor.
	Fd int `json:"fd"`

	// The mac address.
	Mac string `json:"mac"`

	// Size of vnet header expected by the tap device.
	Vnet int `json:"vnet"`

	// Hardware offloads supported by tap device?
	Offload bool `json:"offload"`
}

func (device *VirtioNetDevice) processPackets(
	vchannel *VirtioChannel,
	recv bool) error {

	for buf := range vchannel.incoming {

		header := buf.Map(0, VirtioNetHeaderSize)

		// Legit?
		if len(header) < VirtioNetHeaderSize {
			vchannel.outgoing <- buf
			continue
		}

		// Should we pass the virtio net header to the tap device as the vnet
		// header or strip it off?
		pktStart := VirtioNetHeaderSize - device.Vnet
		pktEnd := buf.Length() - pktStart

		// Doing send or recv?
		if recv {
			buf.Read(device.Fd, pktStart, pktEnd)
		} else {
			buf.Write(device.Fd, pktStart, pktEnd)
		}

		// Done.
		vchannel.outgoing <- buf
	}

	return nil
}

func NewVirtioMmioNet(info *DeviceInfo) (Device, error) {
	device, err := NewMmioVirtioDevice(info, VirtioTypeNet)
	device.Channels[0] = NewVirtioChannel(0, 256)
	device.Channels[1] = NewVirtioChannel(1, 256)
	return &VirtioNetDevice{VirtioDevice: device}, err
}

func NewVirtioPciNet(info *DeviceInfo) (Device, error) {
	device, err := NewPciVirtioDevice(info, PciClassNetwork, VirtioTypeNet, 16)
	device.Channels[0] = NewVirtioChannel(0, 256)
	device.Channels[1] = NewVirtioChannel(1, 256)
	return &VirtioNetDevice{VirtioDevice: device}, err
}

func (nic *VirtioNetDevice) Attach(vm *platform.Vm, model *Model) error {
	if nic.Vnet != 0 && nic.Vnet != VirtioNetHeaderSize {
		return VirtioUnsupportedVnetHeader
	}

	if nic.Vnet > 0 && nic.Offload {
		nic.Debug("hw offloads available, exposing features to guest.")
		nic.SetFeatures(VirtioNetFCsum | VirtioNetFHostTso4 | VirtioNetFHostTso6 |
			VirtioNetFHostEcn | VirtioNetFHostUfo)
	}

	// Set up our Config space.
	nic.Config.GrowTo(VirtioNetConfigLen)

	// Add MAC, if specified. If unspecified or bad
	// autogenerate.
	var mac net.HardwareAddr
	if nic.Mac != "" {
		var err error
		mac, err = net.ParseMAC(nic.Mac)
		if err != nil {
			return err
		}
	} else {
		// Random MAC with Gridcentric's OUI.
		mac = make([]byte, 6)
		rand.Read(mac[3:])
		mac[0] = 0x28
		mac[1] = 0x48
		mac[2] = 0x46
	}
	nic.SetFeatures(VirtioNetFMac)
	for i := 0; i < len(mac); i += 1 {
		nic.Config.Set8(VirtioNetMacOffset+i, mac[i])
	}

	// Add status bits. In the future we should
	// be polling the underlying physical/tap device
	// for link-up and announce status. For now,
	// just emulate the status-less "always up" behavior.
	nic.SetFeatures(VirtioNetFStatus)
	nic.Config.Set16(VirtioNetStatusOffset, VirtioNetLinkUp)

	err := nic.VirtioDevice.Attach(vm, model)
	if err != nil {
		return err
	}

	// Start our network process.
	go nic.processPackets(nic.Channels[0], true)
	go nic.processPackets(nic.Channels[1], false)

	return nil
}
