package machine

import (
    "os"
)

type VirtioConsoleDevice struct {
    *VirtioDevice

    // The backing server fd.
    Fd  int `json:"fd"`
}

func dumpConsole(channel chan []VirtioBuffer) {
    // NOTE: This is just an example of how to
    // use a virtio device for the moment. This
    // will be done more rigorously shortly.
    for bufs := range channel {
        for _, buf := range bufs {
            os.Stdout.Write(buf.data)
        }
    }
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, []uint{16, 16}, VirtioTypeConsole)
    go dumpConsole(device.channels[0].incoming)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, []uint{16, 16}, PciClassMisc, VirtioTypeConsole)
    go dumpConsole(device.channels[0].incoming)
    return &VirtioConsoleDevice{VirtioDevice: device}, err
}
