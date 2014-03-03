package machine

import (
    "novmm/platform"
)

//
// Model --
//
// Our basic machine model.
//
// This is very much different from a standard virtual machine.
// First, we only support a very limited selection of devices.
// We actually do not support *any* I/O-port based devices, which
// includes PCI devices (which require an I/O port at the root).

type Model struct {

    // Basic memory layout:
    // This is generally accessible from the loader,
    // and other modules that may need to tweak memory.
    MemoryMap

    // Basic interrupt layout:
    // This maps interrupts to devices.
    InterruptMap

    // How many vcpus?
    vcpus uint

    // All devices.
    devices []Device

    // Our device lookup cache.
    pio_cache  *IoCache
    mmio_cache *IoCache
}

func NewModel(vm *platform.Vm) (*Model, error) {

    // Create our model object.
    model := new(Model)

    // Setup the memory map.
    model.MemoryMap = make(MemoryMap, 0, 0)

    // Setup the interrupt map.
    model.InterruptMap = make(InterruptMap)

    // Create our devices.
    model.devices = make([]Device, 0, 0)

    // We're set.
    return model, nil
}

func (model *Model) flush() error {

    collectIoHandlers := func(is_pio bool) []IoHandlers {
        io_handlers := make([]IoHandlers, 0, 0)
        for _, device := range model.devices {
            if is_pio {
                io_handlers = append(io_handlers, device.PioHandlers())
            } else {
                io_handlers = append(io_handlers, device.MmioHandlers())
            }
        }
        return io_handlers
    }

    // (Re-)Create our IoCache.
    model.pio_cache = NewIoCache(collectIoHandlers(true), true)
    model.mmio_cache = NewIoCache(collectIoHandlers(false), false)

    // We're okay.
    return nil
}

func (model *Model) Devices() []Device {
    return model.devices
}
