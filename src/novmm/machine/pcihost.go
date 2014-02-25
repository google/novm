package machine

//
// PciHostBridge --
//
// This is an example of PCI-to-PCI bridge.
// It's not particularly well though-out, or sensible.
// We'd probably want to chain a second PCI bus device
// behind it somehow.
//
// As it stands, we're not going to use this bridge.
// AFAIK the reason the CPU doesn't access all PCI devices
// directly is because there is an electrical limit on
// building PCI buses with too many devices. But this is
// a VMM, so we don't have any electrical limits. :)
//

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

    // A bridge only has 2 bars.
    hostbridge.PciBarCount = 2

    // Set our type & command.
    hostbridge.Config[0xe] = 1
    hostbridge.Config[0x4] |= 0x04

    // Add our PortRoot capability.
    hostbridge.Capabilities[PciCapabilityPortRoot] = &PciCapability{
        Size: 0,
    }

    // Done.
    return hostbridge, nil
}
