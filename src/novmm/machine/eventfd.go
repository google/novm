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
)

type WriteIoEvent struct {
	size uint
	data uint64
}

func (event *WriteIoEvent) Size() uint {
	return event.size
}

func (event *WriteIoEvent) GetData() uint64 {
	return event.data
}

func (event *WriteIoEvent) SetData(val uint64) {
	// This really shouldn't happen.
	// Perhaps we should consider recording
	// this and raising an error later?
}

func (event *WriteIoEvent) IsWrite() bool {
	return true
}

func (cache *IoCache) save(
	vm *platform.Vm,
	addr platform.Paddr,
	handler *IoHandler,
	ioevent IoEvent,
	offset uint64) error {

	// Do we have sufficient hits?
	if cache.hits[addr] < 100 {
		return nil
	}

	// Bind an eventfd.
	// Note that we pass in the exactly address here,
	// not the address associated with the IOHandler.
	boundfd, err := vm.NewBoundEventFd(
		addr,
		ioevent.Size(),
		cache.is_pio,
		true,
		ioevent.GetData())
	if err != nil || boundfd == nil {
		return err
	}

	// Create a fake event.
	// This is because the real event will actually
	// reach into the vcpu registers to get the data.
	fake_event := &WriteIoEvent{ioevent.Size(), ioevent.GetData()}

	// Run our function.
	go func(ioevent IoEvent) {

		for {
			// Wait for the next event.
			_, err := boundfd.Wait()
			if err != nil {
				break
			}

			// Call our function.
			// We keep handling this device the same
			// way until it tells us to stop by returning
			// anything other than the SaveIO error.
			err = handler.queue.Submit(ioevent, offset)
			if err != SaveIO {
				break
			}
		}

		// Finished with the eventfd.
		boundfd.Close()

	}(fake_event)

	// Success.
	return nil
}
