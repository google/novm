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
    hostbridge.config[0x6] = 0x10 // Caps present.
    hostbridge.config[0xe] = 1    // Type.

    // Add our capabilities.
    hostbridge.config[0x34] = 0x40                            // Cap pointer.
    hostbridge.config = append(hostbridge.config, byte(0x40)) // Type port root.
    hostbridge.config = append(hostbridge.config, byte(0))    // End of cap pointer.

    // Done.
    return hostbridge, nil
}
