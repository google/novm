package machine

import (
    "log"
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

    // All devices.
    devices []*Device

    // Our device lookup cache.
    pio_cache  *IoCache
    mmio_cache *IoCache

    // User memory to add.
    // This is added as the final step before running,
    // or may be added implicitly by the loader when
    // generating the memory map.
    memory uint64
}

func (model *Model) layoutMemory() error {

    // Try to place our user memory.
    // NOTE: This will be called after all devices
    // have reserved appropriate memory regions, so
    // we will not conflict with anything else.
    last_top := platform.Paddr(0)

    for i := 0; i < len(model.MemoryMap.regions); {

        region := model.MemoryMap.regions[i]

        log.Printf("memory: region [%x,%x)",
            region.Start,
            region.Start.After(region.Size))

        if last_top != region.Start {

            log.Printf("memory: gap @ %x", last_top)

            todo := uint64(region.Start) - uint64(last_top)
            if todo > model.memory {
                todo = model.memory
                model.memory = 0
            } else {
                model.memory -= todo
            }

            _, _, err := model.Allocate(
                User,
                "user memory",
                last_top,
                todo,
                last_top,
                platform.PageSize)
            if err != nil {
                return err
            }

            // This is an insertion.
            // We can safely skip ahead.
            i += 2

        } else {
            // Go to the next region.
            i += 1
        }

        // Remember the top of this region.
        last_top = region.Start.After(region.Size)
    }

    if model.memory > 0 {
        _, _, err := model.Allocate(
            User,
            "user memory",
            last_top,
            model.memory,
            last_top,
            platform.PageSize)
        if err != nil {
            return err
        }

        // Nothing left.
        model.memory = 0
    }

    // All is good.
    return nil
}

func (model *Model) Devices() []*Device {
    return model.devices
}

func (model *Model) Regions() ([]*TypedMemoryRegion, error) {

    if model.memory > 0 {
        // Ensure all user-memory is available.
        err := model.layoutMemory()
        if err != nil {
            return nil, err
        }
    }

    // Return all memory regions.
    return model.MemoryMap.regions, nil
}

func (model *Model) AllocateInterrupt() int {
    // This is currently broken. (Duh).
    // We need to figure out the semantics around
    // direct access virtio-mmio devices and support
    // them properly (assuming the x86 kernel does so).
    return 32
}

func NewModel(
    vm platform.Vm,
    vcpus uint,
    memory uint64) (*Model, error) {

    // Create our model object.
    model := new(Model)

    // Setup the memory map.
    model.MemoryMap.regions = make([]*TypedMemoryRegion, 0, 0)
    model.MemoryMap.vm = vm

    // Reserve our basic "BIOS" memory.
    // This is done simply to match expectations.
    _, _, err := model.MemoryMap.Allocate(
        Reserved,
        "bios",
        0,                 // Start.
        platform.PageSize, // Size.
        0,                 // End.
        platform.PageSize, // Alignment.
    )
    if err != nil {
        return nil, err
    }

    // Allocate and load our ACPI memory.
    err = model.loadAcpi(vcpus)
    if err != nil {
        return nil, err
    }

    // Create our devices.
    model.devices = make([]*Device, 0, 0)

    // Save user memory to install.
    // This happens right at Start().
    model.memory = memory

    // We're set.
    return model, nil
}

func (model *Model) flush() error {

    // Create our catchall device.
    //
    // This will always be the last device in the list,
    // and exists simply to log an error message when the
    // instance access invalid memory. In theory, it could
    // throw errors or have any other arbitrary outcome, but
    // that's not a real machine does (it just ignores it).
    invalid, err := model.NewInvalidDevice()
    if err != nil {
        return err
    }

    // Add our invalid device.
    devices := make(
        []*Device,
        len(model.devices)+1,
        len(model.devices)+1)
    for i, device := range model.devices {
        devices[i] = device
    }
    devices[len(model.devices)] = invalid

    collectIoHandlers := func(is_mmio bool) []*IoHandlers {
        io_handlers := make([]*IoHandlers, 0, 0)

        for _, device := range devices {
            if is_mmio {
                io_handlers = append(io_handlers, &device.Mmio)
                for port_region, _ := range device.Mmio {
                    log.Printf("model: mmio device %s @ [%x,%x)",
                        device.info.Name,
                        port_region.Start,
                        port_region.Start.After(port_region.Size))
                }
            } else {
                io_handlers = append(io_handlers, &device.Pio)
                for port_region, _ := range device.Pio {
                    log.Printf("model: pio device %s @ [%x,%x)",
                        device.info.Name,
                        port_region.Start,
                        port_region.Start.After(port_region.Size))
                }
            }
        }

        return io_handlers
    }

    // (Re-)Create our IoCache.
    model.pio_cache = NewIoCache(collectIoHandlers(false))
    model.mmio_cache = NewIoCache(collectIoHandlers(true))

    // We're okay.
    return nil
}

func (model *Model) Start() error {

    if model.memory > 0 {
        // Ensure all user-memory is available.
        err := model.layoutMemory()
        if err != nil {
            return err
        }
    }

    // Map all memory.
    return model.Map()
}

func (model *Model) Stop() error {

    // Unmap all memory.
    return model.Unmap()
}
