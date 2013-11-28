package machine

func LoadVirtioMmioConsole(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadMmioVirtioDevice(
        model,
        info,
        VirtioTypeConsole)

    return err
}

func LoadVirtioPciConsole(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadPciVirtioDevice(
        model,
        info,
        PciClassMisc,
        VirtioTypeConsole)

    return err
}
