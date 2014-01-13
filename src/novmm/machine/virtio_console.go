package machine

type VirtioConsoleDevice struct {
    *VirtioDevice

    // The backing server fd.
    Fd  int `json:"fd"`
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, []uint{1024, 1024}, VirtioTypeConsole)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, []uint{1024, 1024}, PciClassMisc, VirtioTypeConsole)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}
