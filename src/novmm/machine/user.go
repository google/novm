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
	"sort"
	"syscall"
)

//
// UserMemorySegment --
//
// This maps a file offset to a segment of user memory.
//
type UserMemorySegment struct {
	// The offset in the file.
	Offset uint64 `json:"offset"`

	// The segment of virtual machine memory.
	Region MemoryRegion `json:"region"`
}

//
// UserMemory --
//
// The user memory device allocates and maintains a
// mapping of user memory to a backing file. The file
// can be of any type (i.e. opening /dev/zero or any
// other temporary file) and we rely on the management
// stack to determine the best way to provide memory.
//
type UserMemory struct {
	BaseDevice

	// As laid-out.
	// This is indexed by offset in the file,
	// and each offset points to the given region.
	Allocated []UserMemorySegment `json:"allocated"`

	// The offset in the file.
	Offset int64 `json:"offset"`

	// The FD to map for regions.
	Fd int `json:"fd"`

	// Our map.
	mmap []byte
}

func (user *UserMemory) Reload(
	vm *platform.Vm,
	model *Model) (uint64, uint64, error) {

	total := uint64(0)
	max_offset := uint64(0)

	// Place existing user memory.
	for _, segment := range user.Allocated {

		// Allocate it in the machine.
		err := model.Reserve(
			vm,
			user,
			MemoryTypeUser,
			segment.Region.Start,
			segment.Region.Size,
			user.mmap[segment.Offset:segment.Offset+segment.Region.Size])
		if err != nil {
			return total, max_offset, err
		}

		// Is our max_offset up to date?
		if segment.Offset+segment.Region.Size > max_offset {
			max_offset = segment.Offset + segment.Region.Size
		}

		// Done some more.
		total += segment.Region.Size
	}

	return total, max_offset, nil
}

func (user *UserMemory) Layout(
	vm *platform.Vm,
	model *Model,
	start uint64,
	memory uint64) error {

	// Try to place our user memory.
	// NOTE: This will be called after all devices
	// have reserved appropriate memory regions, so
	// we will not conflict with anything else.
	last_top := platform.Paddr(0)

	sort.Sort(&model.MemoryMap)

	for i := 0; i < len(model.MemoryMap) && memory > 0; i += 1 {

		region := model.MemoryMap[i]

		if last_top != region.Start {

			// How much can we do here?
			gap := uint64(region.Start) - uint64(last_top)
			if gap > memory {
				gap = memory
				memory = 0
			} else {
				memory -= gap
			}

			user.Debug(
				"physical [%x,%x] -> file [%x,%x]",
				last_top, gap-1,
				start, start+gap-1)

			// Allocate the bits.
			err := model.Reserve(
				vm,
				user,
				MemoryTypeUser,
				last_top,
				gap,
				user.mmap[start:start+gap])
			if err != nil {
				return err
			}

			// Remember this.
			user.Allocated = append(
				user.Allocated,
				UserMemorySegment{
					start,
					MemoryRegion{last_top, gap}})

			// Move ahead in the backing store.
			start += gap
		}

		// Remember the top of this region.
		last_top = region.Start.After(region.Size)
	}

	if memory > 0 {
		err := model.Reserve(
			vm,
			user,
			MemoryTypeUser,
			last_top,
			memory,
			user.mmap[start:])
		if err != nil {
			return err
		}
	}

	// All is good.
	return nil
}

func NewUserMemory(info *DeviceInfo) (Device, error) {

	// Create our user memory.
	// Nothing special, no defaults.
	user := new(UserMemory)
	user.Allocated = make([]UserMemorySegment, 0, 0)

	return user, user.init(info)
}

func (user *UserMemory) Attach(vm *platform.Vm, model *Model) error {

	// Create a mmap'ed region.
	var stat syscall.Stat_t
	err := syscall.Fstat(user.Fd, &stat)
	if err != nil {
		return err
	}

	// How big is our memory?
	size := uint64(stat.Size)

	if size > 0 {
		user.mmap, err = syscall.Mmap(
			user.Fd,
			user.Offset,
			int(size),
			syscall.PROT_READ|syscall.PROT_WRITE,
			syscall.MAP_SHARED)
		if err != nil || user.mmap == nil {
			return err
		}
	} else {
		return UserMemoryNotFound
	}

	// Layout the existing regions.
	total, max_offset, err := user.Reload(vm, model)
	if err != nil {
		return err
	}

	// Layout remaining amount.
	if size > total {
		return user.Layout(vm, model, max_offset, size-total)
	}

	return nil
}
