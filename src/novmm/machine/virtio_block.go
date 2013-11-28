package machine

func LoadVirtioMmioBlock(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadMmioVirtioDevice(
        model,
        info,
        VirtioTypeBlock)

    return err
}

func LoadVirtioPciBlock(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadPciVirtioDevice(
        model,
        info,
        PciClassStorage,
        VirtioTypeBlock)

    return err
}
