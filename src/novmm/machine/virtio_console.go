package machine

type VirtioConsoleDevice struct {
    *VirtioDevice

    // The backing server fd.
    Fd  int `json:"fd"`
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeConsole)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioTypeConsole)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}
