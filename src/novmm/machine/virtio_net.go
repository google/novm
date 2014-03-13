package machine

import (
    "novmm/platform"
)

//
// Virtio Net Features
//
const (
    VirtioNetFCsum     uint32 = 1 << 0
    VirtioNetFHostTso4        = 1 << 11
    VirtioNetFHostTso6        = 1 << 12
    VirtioNetFHostEcn         = 1 << 13
    VirtioNetFHostUfo         = 1 << 14
)

const (
    VirtioNetHeaderSize = 10
)

type VirtioNetDevice struct {
    *VirtioDevice

    // The tap device file descriptor.
    Fd  int `json:"fd"`

    // The mac address.
    Mac string `json:"mac"`

    // Size of vnet header expected by the tap device.
    Vnet int `json:"vnet"`

    // Hardware offloads supported by tap device?
    Offload bool `json:"offload"`
}

func (device *VirtioNetDevice) processPackets(
    vchannel *VirtioChannel,
    recv bool) error {

    for buf := range vchannel.incoming {

        header := buf.Map(0, VirtioNetHeaderSize)

        // Legit?
        if header.Size() < VirtioNetHeaderSize {
            vchannel.outgoing <- buf
            continue
        }

        // Should we pass the virtio net header to the tap device as the vnet
        // header or strip it off?
        pktStart := VirtioNetHeaderSize - device.Vnet
        pktEnd := buf.Length() - pktStart

        // Doing send or recv?
        if recv {
            buf.Read(device.Fd, pktStart, pktEnd)
        } else {
            buf.Write(device.Fd, pktStart, pktEnd)
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
    if net.Vnet != 0 && net.Vnet != VirtioNetHeaderSize {
        return VirtioUnsupportedVnetHeader
    }

    if net.Vnet > 0 && net.Offload {
        net.Debug("hw offloads available, exposing features to guest.")
        net.SetFeatures(VirtioNetFCsum | VirtioNetFHostTso4 | VirtioNetFHostTso6 |
            VirtioNetFHostEcn | VirtioNetFHostUfo)
    }

    err := net.VirtioDevice.Attach(vm, model)
    if err != nil {
        return err
    }

    // Start our network process.
    go net.processPackets(net.Channels[0], true)
    go net.processPackets(net.Channels[1], false)

    return nil
}
