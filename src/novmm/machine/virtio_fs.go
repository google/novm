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
	"log"
	"novmm/plan9"
	"novmm/platform"
)

const (
	VirtioFsFMount = 1
)

type VirtioFsDevice struct {
	*VirtioDevice

	// Our filesystem tag.
	Tag string `json:"tag"`

	// Debug fs operations?
	Debugfs bool `json:"debugfs"`

	// Our plan9 server.
	plan9.Fs
}

func (fs *VirtioFsDevice) process(buf *VirtioBuffer) {

	// Map our request & response.
	//
	// NOTE: The way these segments are handled on the wire
	// is pretty odd, in my opinion. We don't touch the original
	// request, but instead append the response to the *next*
	// segment in the list. For "zero-copy" requests (i.e. read),
	// Linux may chain a request segment, data segment, then a
	// response segment. We don't do any looking at the segments
	// here, but rather use the generic Stream() with appropriate
	// offsets. These should map directly and efficiently onto
	// the logical segments, but will handle cases where they
	// don't (for whatever reason).
	//
	// This requires us to peek into the first two bytes of the
	// 9P message and pull out the length. This is the only very
	// 9P-specific code that exists in this module.

	req := NewVirtioStream(buf, 0)
	length := req.Read16()
	req.ReadRewind()
	resp := NewVirtioStream(buf, int(length))

	// Handle our request.
	err := fs.Fs.Handle(req, resp, fs.Debugfs)
	if err != nil {
		log.Printf("FS error: %s", err.Error())
	}

	// Finished request.
	fs.VirtioDevice.Channels[0].outgoing <- buf
}

func (fs *VirtioFsDevice) run() error {

	for {
		// Read a request.
		req := <-fs.VirtioDevice.Channels[0].incoming

		// Process it.
		go fs.process(req)
	}

	return nil
}

func setupFs(device *VirtioDevice) (Device, error) {

	// Create our channel (requests).
	device.Channels[0] = NewVirtioChannel(0, 512)
	device.Channels[1] = NewVirtioChannel(1, 512)

	// Initialize our FS.
	fs := new(VirtioFsDevice)
	fs.VirtioDevice = device
	fs.Tag = "default"

	return fs, fs.Init()
}

func NewVirtioMmioFs(info *DeviceInfo) (Device, error) {
	device, err := NewMmioVirtioDevice(info, VirtioType9p)
	if err != nil {
		return nil, err
	}

	return setupFs(device)
}

func NewVirtioPciFs(info *DeviceInfo) (Device, error) {
	device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioType9p, 16)
	if err != nil {
		return nil, err
	}

	return setupFs(device)
}

func (fs *VirtioFsDevice) Attach(vm *platform.Vm, model *Model) error {
	err := fs.VirtioDevice.Attach(vm, model)
	if err != nil {
		return err
	}

	// This feature is not optional.
	// We ignore serialized features.
	fs.SetFeatures(VirtioFsFMount)

	// Make sure the config reflects our tag.
	tag_bytes := []byte(fs.Tag)
	fs.Config.GrowTo(2 + len(tag_bytes) + 1)
	fs.Config.Set16(0, uint16(len(tag_bytes)))
	for i := 0; i < len(tag_bytes); i += 1 {
		fs.Config.Set8(2+i, uint8(tag_bytes[i]))
	}

	// Ensure the file system is sane.
	err = fs.Fs.Attach()
	if err != nil {
		return err
	}

	// Start our backend process.
	go fs.run()

	return nil
}
