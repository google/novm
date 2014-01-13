package machine

// A driver load function.
type Driver func(info *DeviceInfo) (Device, error)

// All available device drivers.
var drivers = map[string]Driver{
    "tss":                 NewTss,
    "bios":                NewBios,
    "acpi":                NewAcpi,
    "rtc":                 NewRtc,
    "uart":                NewUart,
    "pci-bus":             NewPciBus,
    "pci-hostbridge":      NewPciHostBridge,
    "user-memory":         NewUserMemory,
    "virtio-pci-block":    NewVirtioPciBlock,
    "virtio-mmio-block":   NewVirtioMmioBlock,
    "virtio-pci-console":  NewVirtioPciConsole,
    "virtio-mmio-console": NewVirtioMmioConsole,
    "virtio-pci-net":      NewVirtioPciNet,
    "virtio-mmio-net":     NewVirtioMmioNet,
    "virtio-pci-fs":       NewVirtioPciFs,
    "virtio-mmio-fs":      NewVirtioMmioFs,
}
