package machine

type VirtioConsoleDevice struct {
    *VirtioDevice

    // The backing server fd.
    Fd  int `json:"fd"`
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, []uint{16, 16}, VirtioTypeConsole)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, []uint{16, 16}, PciClassMisc, VirtioTypeConsole)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}
