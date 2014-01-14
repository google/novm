package machine

type VirtioBlockDevice struct {
    *VirtioDevice

    // The backing file.
    Fd  int `json:"fd"`
}

func NewVirtioMmioBlock(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeBlock)
    device.Channels[0] = device.NewVirtioChannel(256)
    return &VirtioBlockDevice{VirtioDevice: device}, err
}

func NewVirtioPciBlock(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassStorage, VirtioTypeBlock)
    device.Channels[0] = device.NewVirtioChannel(256)
    return &VirtioBlockDevice{VirtioDevice: device}, err
}
