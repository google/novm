package machine

import (
    "novmm/platform"
    "os"
)

type VirtioConsoleDevice struct {
    *VirtioDevice

    // The backing server fd.
    Fd  int `json:"fd"`
}

func (device *VirtioConsoleDevice) dumpConsole(vchannel *VirtioChannel) {
    // NOTE: This is just an example of how to
    // use a virtio device for the moment. This
    // will be done more rigorously shortly.
    for bufs := range vchannel.incoming {
        for _, buf := range bufs {
            os.Stdout.Write(buf.data)
        }
        vchannel.outgoing <- bufs
    }
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeConsole)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioTypeConsole)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}

func (console *VirtioConsoleDevice) Attach(vm *platform.Vm, model *Model) error {
    err := console.VirtioDevice.Attach(vm, model)
    if err != nil {
        return err
    }

    // Start our console process.
    go console.dumpConsole(console.Channels[0])

    return nil
}
