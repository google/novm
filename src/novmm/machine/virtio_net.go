package machine

type VirtioNetDevice struct {
    *VirtioDevice

    // The tap device file descriptor.
    Fd  int `json:"fd"`

    // The mac address.
    Mac string `json:"mac"`
}

func NewVirtioMmioNet(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeNet)
    return &VirtioNetDevice{VirtioDevice: device}, err
}

func NewVirtioPciNet(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassNetwork, VirtioTypeNet)
    return &VirtioNetDevice{VirtioDevice: device}, err
}
