package machine

type IoMap map[MemoryRegion]IoOperations
type IoHandlers map[MemoryRegion]*IoHandler

type Device struct {
    // Generic info.
    info *DeviceInfo

    // Port I/O handlers.
    Pio IoHandlers

    // Mmio I/O handlers.
    Mmio IoHandlers
}

func NewDevice(
    info *DeviceInfo,
    pio IoMap,
    pio_offset uint64,
    mmio IoMap,
    mmio_offset uint64) (*Device, error) {

    // Create our device.
    device := new(Device)
    device.info = info

    // Initialize all handlers.
    device.Pio = make(IoHandlers)
    device.Mmio = make(IoHandlers)
    for region, ops := range pio {
        new_region := MemoryRegion{region.Start.After(pio_offset), region.Size}
        device.Pio[new_region] = NewIoHandler(device.info, new_region, ops)
    }
    for region, ops := range mmio {
        // Set our region.
        new_region := MemoryRegion{region.Start.After(mmio_offset), region.Size}
        device.Mmio[new_region] = NewIoHandler(device.info, new_region, ops)
    }

    // We're good.
    return device, nil
}

func (model *Model) AddDevice(device *Device) error {

    model.devices = append(model.devices, device)
    return model.flush()
}
