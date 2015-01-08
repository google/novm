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

//
// PciHostBridge --
//
// This is an example of PCI-to-PCI bridge.
// It's not particularly well though-out, or sensible.
// We'd probably want to chain a second PCI bus device
// behind it somehow.
//
// As it stands, we're not going to use this bridge.
// AFAIK the reason the CPU doesn't access all PCI devices
// directly is because there is an electrical limit on
// building PCI buses with too many devices. But this is
// a VMM, so we don't have any electrical limits. :)
//

const (
	PciCapabilityPortRoot = 0x40
)

func NewPciHostBridge(info *DeviceInfo) (Device, error) {

	// Create a bus device.
	hostbridge, err := NewPciDevice(
		info,
		PciVendorId(0x1022), // AMD.
		PciDeviceId(0x7432), // Made-up.
		PciClassBridge,
		PciRevision(0),
		0,
		0)
	if err != nil {
		return nil, err
	}

	// A bridge only has 2 bars.
	hostbridge.PciBarCount = 2

	// Set our type & command.
	hostbridge.Config.Set8(0xe, 1)
	hostbridge.Config.Set8(0x4, hostbridge.Config.Get8(0x4)|0x04)

	// Add our PortRoot capability.
	hostbridge.Capabilities[PciCapabilityPortRoot] = &PciCapability{
		Size: 0,
	}

	// Done.
	return hostbridge, nil
}
