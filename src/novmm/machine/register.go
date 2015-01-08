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
)

type Register struct {
	// The value of the register.
	Value uint64 `json:"value"`

	// Read-only bits?
	readonly uint64

	// Clear these bits on read.
	readclr uint64
}

func (register *Register) Read(offset uint64, size uint) (uint64, error) {
	var mask uint64

	switch size {
	case 1:
		mask = 0x00000000000000ff
	case 2:
		mask = 0x000000000000ffff
	case 4:
		mask = 0x00000000ffffffff
	case 8:
		mask = 0xffffffffffffffff
	}

	value := uint64(math.MaxUint64)

	switch offset {
	case 0:
		value = (register.Value) & mask
	case 1:
		value = (register.Value >> 8) & mask
		mask = mask << 8
	case 2:
		value = (register.Value >> 16) & mask
		mask = mask << 16
	case 3:
		value = (register.Value >> 24) & mask
		mask = mask << 24
	case 4:
		value = (register.Value >> 32) & mask
		mask = mask << 32
	case 5:
		value = (register.Value >> 40) & mask
		mask = mask << 40
	case 6:
		value = (register.Value >> 48) & mask
		mask = mask << 48
	case 7:
		value = (register.Value >> 56) & mask
		mask = mask << 56
	}

	register.Value = register.Value & ^(mask & register.readclr)
	return value, nil
}

func (register *Register) Write(offset uint64, size uint, value uint64) error {
	var mask uint64

	switch size {
	case 1:
		mask = 0x00000000000000ff & ^register.readonly
	case 2:
		mask = 0x000000000000ffff & ^register.readonly
	case 4:
		mask = 0x00000000ffffffff & ^register.readonly
	case 8:
		mask = 0xffffffffffffffff & ^register.readonly
	}

	value = value & mask

	switch offset {
	case 1:
		mask = mask << 8
		value = value << 8
	case 2:
		mask = mask << 16
		value = value << 16
	case 3:
		mask = mask << 24
		value = value << 24
	case 4:
		mask = mask << 32
		value = value << 32
	case 5:
		mask = mask << 40
		value = value << 40
	case 6:
		mask = mask << 48
		value = value << 48
	case 7:
		mask = mask << 56
		value = value << 56
	}

	register.Value = (register.Value & ^mask) | (value & mask)
	return nil
}
