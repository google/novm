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

// A driver load function.
type Driver func(info *DeviceInfo) (Device, error)

// All available device drivers.
var drivers = map[string]Driver{
	"bios":                NewBios,
	"apic":                NewApic,
	"pit":                 NewPit,
	"acpi":                NewAcpi,
	"rtc":                 NewRtc,
	"clock":               NewClock,
	"uart":                NewUart,
	"pci-bus":             NewPciBus,
	"pci-hostbridge":      NewPciHostBridge,
	"user-memory":         NewUserMemory,
	"virtio-pci-block":    NewVirtioPciBlock,
	"virtio-mmio-block":   NewVirtioMmioBlock,
	"virtio-pci-console":  NewVirtioPciConsole,
	"virtio-mmio-console": NewVirtioMmioConsole,
	"virtio-pci-net":      NewVirtioPciNet,
	"virtio-mmio-net":     NewVirtioMmioNet,
	"virtio-pci-fs":       NewVirtioPciFs,
	"virtio-mmio-fs":      NewVirtioMmioFs,
}
