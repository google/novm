package machine

import (
    "math"
)

type InvalidPort struct {
    // No state.
}

type InvalidAddr struct {
    // No state.
}

func (test *InvalidPort) Read(offset uint64, size uint) (uint64, error) {
    return math.MaxUint64, nil
}

func (test *InvalidPort) Write(offset uint64, size uint, value uint64) error {
    return nil
}

func (test *InvalidAddr) Read(offset uint64, size uint) (uint64, error) {
    return math.MaxUint64, nil
}

func (test *InvalidAddr) Write(offset uint64, size uint, value uint64) error {
    return nil
}

func (model *Model) NewInvalidDevice() (*Device, error) {

    return NewDevice(
        &DeviceInfo{},
        IoMap{
            // Setup a simple covering of all ports.
            MemoryRegion{0x0, math.MaxUint64}: &InvalidPort{},
        },
        0,  // Port-I/O offset.
        IoMap{
            // Similarly, cover a all memory.
            MemoryRegion{0x0, math.MaxUint64}: &InvalidAddr{},
        },
        0,  // Memory-I/O offset.
    )
}
