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
	"novmm/platform"
	"sync"
)

const (
	VirtioConsoleFSize      = 1
	VirtioConsoleFMultiPort = 2
)

const (
	VirtioConsoleDeviceReady = 0
	VirtioConsolePortAdd     = 1
	VirtioConsolePortRemove  = 2
	VirtioConsolePortReady   = 3
	VirtioConsolePortConsole = 4
	VirtioConsolePortResize  = 5
	VirtioConsolePortOpen    = 6
	VirtioConsolePortName    = 7
)

type VirtioConsoleDevice struct {
	*VirtioDevice

	read_buf    *VirtioBuffer
	read_offset int
	read_lock   sync.Mutex

	write_lock sync.Mutex

	Opened bool `json:"opened"`
}

func (device *VirtioConsoleDevice) sendCtrl(
	port int,
	event int,
	value int) error {

	buf := <-device.Channels[2].incoming

	header := &Ram{buf.Map(0, 8)}

	if header.Size() < 8 {
		buf.length = 0
		device.Channels[2].outgoing <- buf
		return nil
	}

	header.Set32(0, uint32(port))
	header.Set16(4, uint16(event))
	header.Set16(6, uint16(value))
	buf.length = 8

	device.Channels[2].outgoing <- buf
	return nil
}

func (device *VirtioConsoleDevice) ctrlConsole(
	vchannel *VirtioChannel) error {

	for buf := range vchannel.incoming {

		header := &Ram{buf.Map(0, 8)}

		// Legit?
		if header.Size() < 8 {
			device.Debug("invalid ctrl packet?")
			vchannel.outgoing <- buf
			continue
		}

		id := header.Get32(0)
		event := header.Get16(4)
		value := header.Get16(6)

		// Return the buffer.
		vchannel.outgoing <- buf

		switch int(event) {
		case VirtioConsoleDeviceReady:
			vchannel.Debug("device-ready")
			device.sendCtrl(0, VirtioConsolePortAdd, 1)
			break

		case VirtioConsolePortAdd:
			vchannel.Debug("port-add?")
			break

		case VirtioConsolePortRemove:
			vchannel.Debug("port-remove?")
			break

		case VirtioConsolePortReady:
			vchannel.Debug("port-ready")

			if id == 0 && value == 1 {
				// No, this is not a console.
				device.sendCtrl(0, VirtioConsolePortConsole, 0)
				device.sendCtrl(0, VirtioConsolePortOpen, 1)
				if !device.Opened {
					device.Opened = true
					device.read_lock.Unlock()
					device.write_lock.Unlock()
				}
			}
			break

		case VirtioConsolePortConsole:
			vchannel.Debug("port-console?")
			break

		case VirtioConsolePortResize:
			vchannel.Debug("port-resize")
			break

		case VirtioConsolePortOpen:
			vchannel.Debug("port-open")
			break

		case VirtioConsolePortName:
			vchannel.Debug("port-name")
			break

		default:
			vchannel.Debug("unknown?")
			break
		}
	}

	return nil
}

func setupConsole(device *VirtioDevice) (Device, error) {

	// Set our features.
	device.SetFeatures(VirtioConsoleFMultiPort)

	// We only support a single port.
	// (The worst multi-port device in history).
	device.Config.GrowTo(8)
	device.Config.Set32(4, 1)

	device.Channels[0] = NewVirtioChannel(0, 128)
	device.Channels[1] = NewVirtioChannel(1, 128)
	device.Channels[2] = NewVirtioChannel(2, 32)
	device.Channels[3] = NewVirtioChannel(3, 32)

	return &VirtioConsoleDevice{
		VirtioDevice: device}, nil
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
	device, err := NewMmioVirtioDevice(info, VirtioTypeConsole)
	if err != nil {
		return nil, err
	}

	return setupConsole(device)
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
	device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioTypeConsole, 16)
	if err != nil {
		return nil, err
	}

	return setupConsole(device)
}

func (console *VirtioConsoleDevice) Attach(vm *platform.Vm, model *Model) error {
	err := console.VirtioDevice.Attach(vm, model)
	if err != nil {
		return err
	}

	if !console.Opened {
		// Ensure no reads/writes go through.
		console.read_lock.Lock()
		console.write_lock.Lock()
	}

	// Start our console process.
	go console.ctrlConsole(console.Channels[3])

	return nil
}

func (console *VirtioConsoleDevice) Read(p []byte) (int, error) {

	console.read_lock.Lock()
	defer console.read_lock.Unlock()

	// Need a new buffer?
	if console.read_buf == nil {
		console.read_buf = <-console.Channels[1].incoming
	}

	// Copy out as much as possible.
	n := console.read_buf.CopyOut(console.read_offset, p)
	console.read_offset += n
	if console.read_offset == console.read_buf.Length() {
		// Done with this buffer.
		console.Channels[1].outgoing <- console.read_buf
		console.read_buf = nil
		console.read_offset = 0
	}

	return n, nil
}

func (console *VirtioConsoleDevice) Write(p []byte) (int, error) {

	console.write_lock.Lock()
	defer console.write_lock.Unlock()

	var n int

	for n < len(p) {

		// Always grab a new buffer.
		buf := <-console.Channels[0].incoming

		// Map as much as needed.
		left := len(p) - n
		data := buf.Map(0, left)
		if len(data) <= left {
			copy(data, p[n:n+len(data)])
			n += len(data)
			buf.length = len(data)
		} else {
			copy(data, p[n:])
			n += left
			buf.length = left
		}

		// Put the buffer back.
		console.Channels[0].outgoing <- buf
	}

	// We're done.
	return n, nil
}

func (console *VirtioConsoleDevice) Close() error {
	// Ignore.
	return nil
}
