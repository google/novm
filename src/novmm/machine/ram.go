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
	"math"
	"unsafe"
)

type Ram struct {
	Data []byte `json:"data"`
}

func (ram *Ram) Size() int {
	return len(ram.Data)
}

func (ram *Ram) GrowTo(size int) {
	if size > len(ram.Data) {
		missing := size - len(ram.Data)
		new_bytes := make([]byte, missing, missing)
		ram.Data = append(ram.Data, new_bytes...)
	}
}

func (ram *Ram) Set8(offset int, data uint8) {
	ram.Data[offset] = byte(data)
}

func (ram *Ram) Get8(offset int) uint8 {
	return ram.Data[offset]
}

func (ram *Ram) Set16(offset int, data uint16) {
	*(*uint16)(unsafe.Pointer(&ram.Data[offset])) = data
}

func (ram *Ram) Get16(offset int) uint16 {
	return *(*uint16)(unsafe.Pointer(&ram.Data[offset]))
}

func (ram *Ram) Set32(offset int, data uint32) {
	*(*uint32)(unsafe.Pointer(&ram.Data[offset])) = data
}

func (ram *Ram) Get32(offset int) uint32 {
	return *(*uint32)(unsafe.Pointer(&ram.Data[offset]))
}

func (ram *Ram) Set64(offset int, data uint64) {
	*(*uint64)(unsafe.Pointer(&ram.Data[offset])) = data
}

func (ram *Ram) Get64(offset int) uint64 {
	return *(*uint64)(unsafe.Pointer(&ram.Data[offset]))
}

func (ram *Ram) Read(offset uint64, size uint) (uint64, error) {

	value := uint64(math.MaxUint64)

	// Is it greater than our size?
	if offset+uint64(size) > uint64(ram.Size()) {
		// Ignore.
		return value, nil
	}

	// Handle default.
	switch size {
	case 1:
		value = uint64(ram.Get8(int(offset)))
	case 2:
		value = uint64(ram.Get16(int(offset)))
	case 4:
		value = uint64(ram.Get32(int(offset)))
	case 8:
		value = ram.Get64(int(offset))
	}

	return value, nil
}

func (ram *Ram) Write(offset uint64, size uint, value uint64) error {

	// Is it greater than our size?
	if offset+uint64(size) > uint64(ram.Size()) {
		// Ignore.
		return nil
	}

	// Handle default.
	switch size {
	case 1:
		ram.Set8(int(offset), uint8(value))
	case 2:
		ram.Set16(int(offset), uint16(value))
	case 4:
		ram.Set32(int(offset), uint32(value))
	case 8:
		ram.Set64(int(offset), value)
	}

	return nil
}

func NewRam(size int) *Ram {
	ram := new(Ram)
	ram.Data = make([]byte, size, size)
	return ram
}
