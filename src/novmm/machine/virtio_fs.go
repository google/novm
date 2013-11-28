package machine

func LoadVirtioMmioFs(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadMmioVirtioDevice(
        model,
        info,
        VirtioType9p)

    return err
}

func LoadVirtioPciFs(
    model *Model,
    info *DeviceInfo) error {

    _, err := LoadPciVirtioDevice(
        model,
        info,
        PciClassMisc,
        VirtioType9p)

    return err
}
