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
	"unsafe"
)

type MemoryType int

const (
	MemoryTypeReserved MemoryType = iota
	MemoryTypeUser                = 1
	MemoryTypeAcpi                = 2
	MemoryTypeSpecial             = 3
)

//
// MemoryRegion --
//
// This is a serializable type. It is basically
// a specification for a region of memory, but it
// doesn't carry any information about what type
// of region it should be (or the owner). This is
// used to track the runtime state of the model,
// and will be recreated from state on resume.
//
type MemoryRegion struct {
	Start platform.Paddr `json:"start"`
	Size  uint64         `json:"size"`
}

//
// TypedMemoryRegion --
//
// This is used for tracking runtime state.
// These TypedMemoryRegion's will be created and
// tracked in a MemoryMap (below) within the model,
// and relate directly to runtime platform state.
//
type TypedMemoryRegion struct {
	MemoryRegion
	MemoryType

	// The owner.
	Device

	// The memory pointer (slice).
	user []byte

	// Allocated chunks.
	// These are offsets, which point
	// to the amount of memory allocated.
	allocated map[uint64]uint64
}

//
// MemoryMap --
//
// Our collection of current memory regions.
//
type MemoryMap []*TypedMemoryRegion

func (region *MemoryRegion) End() platform.Paddr {
	return region.Start.After(region.Size)
}

func (region *MemoryRegion) Overlaps(start platform.Paddr, size uint64) bool {
	return ((region.Start >= start && region.Start < start.After(size)) ||
		(region.End() > start && region.End() <= start.After(size)))
}

func (region *MemoryRegion) Contains(start platform.Paddr, size uint64) bool {
	return region.Start <= start && region.End() >= start.After(size)
}

func (memory *MemoryMap) Len() int {
	return len(*memory)
}

func (memory *MemoryMap) Swap(i int, j int) {
	(*memory)[i], (*memory)[j] = (*memory)[j], (*memory)[i]
}

func (memory *MemoryMap) Less(i int, j int) bool {
	return (*memory)[i].Start < (*memory)[j].Start
}

func (memory *MemoryMap) Conflicts(start platform.Paddr, size uint64) bool {
	for _, orig_region := range *memory {
		if orig_region.Overlaps(start, size) {
			return true
		}
	}
	return false
}

func (memory *MemoryMap) Add(region *TypedMemoryRegion) error {
	if memory.Conflicts(region.Start, region.Size) {
		return MemoryConflict
	}

	*memory = append(*memory, region)
	sort.Sort(memory)
	return nil
}

func (memory *MemoryMap) Max() platform.Paddr {
	if len(*memory) == 0 {
		// No memory available?
		return platform.Paddr(0)
	}

	// Return the highest available address.
	top := (*memory)[len(*memory)-1]
	return top.End()
}

func (memory *MemoryMap) Reserve(
	vm *platform.Vm,
	device Device,
	memtype MemoryType,
	start platform.Paddr,
	size uint64,
	user []byte) error {

	// Verbose messages.
	device.Debug(
		"reserving (type: %d) of size %x in [%x,%x]",
		memtype,
		size,
		start,
		start.After(size-1))

	// Ensure all targets are aligned.
	if (start.Align(platform.PageSize, false) != start) ||
		(size%platform.PageSize != 0) {
		return MemoryUnaligned
	}

	// Ensure underlying map is aligned.
	// This may be harder to detect later on.
	if user != nil &&
		uintptr(unsafe.Pointer(&user[0]))%platform.PageSize != 0 {
		return MemoryUnaligned
	}

	// Add the region.
	region := &TypedMemoryRegion{
		MemoryRegion: MemoryRegion{start, size},
		MemoryType:   memtype,
		Device:       device,
		user:         user,
		allocated:    make(map[uint64]uint64),
	}
	err := memory.Add(region)
	if err != nil {
		return err
	}

	// Do the mapping.
	switch region.MemoryType {
	case MemoryTypeUser:
		err = vm.MapUserMemory(region.Start, region.Size, region.user)
	case MemoryTypeReserved:
		err = vm.MapReservedMemory(region.Start, region.Size)
	case MemoryTypeAcpi:
		err = vm.MapUserMemory(region.Start, region.Size, region.user)
	case MemoryTypeSpecial:
		err = vm.MapSpecialMemory(region.Start)
	}

	return err
}

func (memory *MemoryMap) Map(
	memtype MemoryType,
	addr platform.Paddr,
	size uint64,
	allocate bool) ([]byte, error) {

	for i := 0; i < len(*memory); i += 1 {

		region := (*memory)[i]

		if region.Contains(addr, size) &&
			region.MemoryType == memtype {

			addr_offset := uint64(addr - region.Start)

			if allocate {
				// Mark it as used.
				for offset, alloc_size := range region.allocated {
					if (addr_offset >= offset &&
						addr_offset < offset+alloc_size) ||
						(addr_offset+size >= offset &&
							addr_offset < offset) {

						// Already allocated?
						return nil, MemoryConflict
					}
				}

				// Found it.
				region.allocated[addr_offset] = size
			}

			if region.user != nil {
				return region.user[addr_offset : addr_offset+size], nil
			} else {
				return nil, nil
			}
		}
	}

	return nil, MemoryNotFound
}

func (memory *MemoryMap) Allocate(
	memtype MemoryType,
	start platform.Paddr,
	end platform.Paddr,
	size uint64,
	top bool) (platform.Paddr, []byte, error) {

	if top {
		for ; end >= start; end -= platform.PageSize {

			mmap, _ := memory.Map(memtype, end, size, true)
			if mmap != nil {
				return end, mmap, nil
			}
		}

	} else {
		for ; start <= end; start += platform.PageSize {

			mmap, _ := memory.Map(memtype, start, size, true)
			if mmap != nil {
				return start, mmap, nil
			}
		}
	}

	// Couldn't find available memory.
	return platform.Paddr(0), nil, MemoryNotFound
}

func (memory *MemoryMap) Load(
	start platform.Paddr,
	end platform.Paddr,
	data []byte,
	top bool) (platform.Paddr, error) {

	// Allocate the backing data.
	addr, backing_mmap, err := memory.Allocate(
		MemoryTypeUser,
		start,
		end,
		uint64(len(data)),
		top)
	if err != nil {
		return platform.Paddr(0), err
	}

	// Copy it in.
	copy(backing_mmap, data)

	// We're good.
	return addr, nil
}
