package machine

import (
    "math"
)

type Register struct {
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
        mask = 0x000000ff
    case 2:
        mask = 0x0000ffff
    case 3:
        mask = 0x00ffffff
    case 4:
        mask = 0xffffffff
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
    }

    register.Value = register.Value & ^(mask & register.readclr)
    return value, nil
}

func (register *Register) Write(offset uint64, size uint, value uint64) error {
    var mask uint64

    switch size {
    case 1:
        mask = 0x000000ff & ^register.readonly
    case 2:
        mask = 0x0000ffff & ^register.readonly
    case 3:
        mask = 0x00ffffff & ^register.readonly
    case 4:
        mask = 0xffffffff & ^register.readonly
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
    }

    register.Value = (register.Value & ^mask) | (value & mask)
    return nil
}
