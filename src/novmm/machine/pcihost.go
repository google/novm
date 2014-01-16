package machine

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
    hostbridge.Config[0x6] = 0x10 // Caps present.
    hostbridge.Config[0xe] = 1    // Type.

    // Add our capabilities.
    hostbridge.Config.GrowTo(0x42)
    hostbridge.Config[0x34] = 0x40 // Cap pointer.
    hostbridge.Config[0x40] = 0x40 // Type port root.
    hostbridge.Config[0x41] = 0x0  // End of cap pointer.

    // Done.
    return hostbridge, nil
}
