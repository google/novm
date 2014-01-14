package machine

type VirtioFsDevice struct {
    *VirtioDevice

    // The read mappings.
    Read map[string]string

    // The write mappings.
    Write map[string]string
}

func NewVirtioMmioFs(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioType9p)
    device.Channels[0] = device.NewVirtioChannel(16)
    device.Channels[1] = device.NewVirtioChannel(16)
    return &VirtioFsDevice{VirtioDevice: device}, err
}

func NewVirtioPciFs(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioType9p)
    device.Channels[0] = device.NewVirtioChannel(16)
    device.Channels[1] = device.NewVirtioChannel(16)
    return &VirtioFsDevice{VirtioDevice: device}, err
}
