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
	"errors"
	"fmt"
)

// Basic errors.
var DeviceAlreadyPaused = errors.New("Device already paused!")
var DeviceNotPaused = errors.New("Device not paused!")

// Memory allocation / layout errors.
var MemoryConflict = errors.New("Memory regions conflict!")
var MemoryNotFound = errors.New("Memory region not found!")
var MemoryBusy = errors.New("Memory could not be allocated!")
var MemoryUnaligned = errors.New("Memory not aligned!")
var UserMemoryNotFound = errors.New("No user memory found?")

// Interrupt allocation errors.
var InterruptConflict = errors.New("Device interrupt conflict!")
var InterruptUnavailable = errors.New("No interrupt available!")

// PCI errors.
var PciInvalidAddress = errors.New("Invalid PCI address!")
var PciBusNotFound = errors.New("Requested PCI devices, but no bus found?")
var PciMSIError = errors.New("MSI internal error?")
var PciCapabilityMismatch = errors.New("Capability mismatch!")

// UART errors.
var UartUnknown = errors.New("Unknown COM port.")

// Driver errors.
func DriverUnknown(name string) error {
	return errors.New(fmt.Sprintf("Unknown driver: %s", name))
}

// Virtio errors.
var VirtioInvalidQueueSize = errors.New("Invalid VirtIO queue size!")
var VirtioUnsupportedVnetHeader = errors.New("Unsupported vnet header size.")

// I/O memoize errors.
// This is an internal-only error which is returned from
// a write handler. When this is returned (and the cache
// has had a significant number of hits at that address)
// we will create an eventfd for that particular address
// and value. This will reduce the number of kernel-user
// switches necessary to handle that particular address.
var SaveIO = errors.New("Save I/O request (internal error).")
