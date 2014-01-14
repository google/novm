package machine

type VirtioFsDevice struct {
    *VirtioDevice

    // The read mappings.
    Read map[string]string

    // The write mappings.
    Write map[string]string
}

func NewVirtioMmioFs(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, []uint{256, 256}, VirtioType9p)
    return &VirtioFsDevice{VirtioDevice: device}, err
}

func NewVirtioPciFs(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, []uint{256, 256}, PciClassMisc, VirtioType9p)
    return &VirtioFsDevice{VirtioDevice: device}, err
}
