package machine

import (
    "novmm/platform"
)

type VirtioNetDevice struct {
    *VirtioDevice

    // The tap device file descriptor.
    Fd  int `json:"fd"`

    // The mac address.
    Mac string `json:"mac"`
}

func (device *VirtioNetDevice) processPackets(
    vchannel *VirtioChannel,
    recv bool) error {

    for buf := range vchannel.incoming {

        header := buf.Map(0, 10)

        // Legit?
        if header.Size() < 10 {
            vchannel.outgoing <- buf
            continue
        }

        // Doing send or recv?
        if recv {
            buf.Read(device.Fd, 10, buf.Length()-10)
        } else {
            buf.Write(device.Fd, 10, buf.Length()-10)
        }

        // Done.
        vchannel.outgoing <- buf
    }

    return nil
}

func NewVirtioMmioNet(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeNet)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    return &VirtioNetDevice{VirtioDevice: device}, err
}

func NewVirtioPciNet(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassNetwork, VirtioTypeNet, 16)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    return &VirtioNetDevice{VirtioDevice: device}, err
}

func (net *VirtioNetDevice) Attach(vm *platform.Vm, model *Model) error {
    err := net.VirtioDevice.Attach(vm, model)
    if err != nil {
        return err
    }

    // Start our network process.
    go net.processPackets(net.Channels[0], true)
    go net.processPackets(net.Channels[1], false)

    return nil
}
