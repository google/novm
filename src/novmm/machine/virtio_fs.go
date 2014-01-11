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
    return &VirtioFsDevice{VirtioDevice: device}, err
}

func NewVirtioPciFs(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioType9p)
    return &VirtioFsDevice{VirtioDevice: device}, err
}
