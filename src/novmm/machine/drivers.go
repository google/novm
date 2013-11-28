package machine

// A driver load function.
type DeviceAdder func(model *Model, info *DeviceInfo) error

// All available device drivers.
var drivers = map[string]DeviceAdder{
    "rtc":                 LoadRtc,
    "uart":                LoadUart,
    "pci-bus":             LoadPciBus,
    "virtio-pci-block":    LoadVirtioPciBlock,
    "virtio-mmio-block":   LoadVirtioMmioBlock,
    "virtio-pci-console":  LoadVirtioPciConsole,
    "virtio-mmio-console": LoadVirtioMmioConsole,
    "virtio-pci-net":      LoadVirtioPciNet,
    "virtio-mmio-net":     LoadVirtioMmioNet,
    "virtio-pci-fs":       LoadVirtioPciFs,
    "virtio-mmio-fs":      LoadVirtioMmioFs,
}

func (model *Model) LoadDevice(info *DeviceInfo) error {

    // Create a new device info.
    new_info := new(DeviceInfo)
    new_info.Name = info.Name
    new_info.Data = info.Data
    new_info.Debug = info.Debug

    // Find the appropriate driver.
    load_fn, ok := drivers[info.Driver]
    if !ok {
        return DriverUnknown(info.Driver)
    }

    // Try adding the device.
    return load_fn(model, new_info)
}
