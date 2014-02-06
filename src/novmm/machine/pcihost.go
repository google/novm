package machine

const (
    PciCapabilityPortRoot = 0x40
)

func NewPciHostBridge(info *DeviceInfo) (Device, error) {

    // Create a bus device.
    hostbridge, err := NewPciDevice(
        info,
        PciVendorId(0x1022), // AMD.
        PciDeviceId(0x7432), // Made-up.
        PciClassBridge,
        PciRevision(0),
        0,
        0)
    if err != nil {
        return nil, err
    }

    // Set our type.
    hostbridge.Config[0xe] = 1

    // Add our PortRoot capability.
    hostbridge.Capabilities[PciCapabilityPortRoot] = &PciCapability{
        Size: 0,
    }

    // Done.
    return hostbridge, nil
}
