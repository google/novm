package machine

func LoadVirtioMmioNet(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadMmioVirtioDevice(
        model,
        info,
        VirtioTypeNet)

    return err
}

func LoadVirtioPciNet(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadPciVirtioDevice(
        model,
        info,
        PciClassNetwork,
        VirtioTypeNet)

    return err
}
