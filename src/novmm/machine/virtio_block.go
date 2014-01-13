package machine

type VirtioBlockDevice struct {
    *VirtioDevice

    // The backing file.
    Fd  int `json:"fd"`
}

func NewVirtioMmioBlock(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, []uint{1024}, VirtioTypeBlock)
    return &VirtioBlockDevice{VirtioDevice: device}, err
}

func NewVirtioPciBlock(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, []uint{1024}, PciClassStorage, VirtioTypeBlock)
    return &VirtioBlockDevice{VirtioDevice: device}, err
}
