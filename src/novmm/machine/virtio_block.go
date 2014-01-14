package machine

type VirtioBlockDevice struct {
    *VirtioDevice

    // The backing file.
    Fd  int `json:"fd"`
}

func NewVirtioMmioBlock(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, []uint{256}, VirtioTypeBlock)
    return &VirtioBlockDevice{VirtioDevice: device}, err
}

func NewVirtioPciBlock(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, []uint{256}, PciClassStorage, VirtioTypeBlock)
    return &VirtioBlockDevice{VirtioDevice: device}, err
}
