package machine

import (
    "novmm/platform"
    "syscall"
)

const (
    VirtioConsoleFSize      = 1
    VirtioConsoleFMultiPort = 2
)

const (
    VirtioConsoleDeviceReady = 0
    VirtioConsolePortAdd     = 1
    VirtioConsolePortRemove  = 2
    VirtioConsolePortReady   = 3
    VirtioConsolePortConsole = 4
    VirtioConsolePortResize  = 5
    VirtioConsolePortOpen    = 6
    VirtioConsolePortName    = 7
)

type VirtioConsoleDevice struct {
    *VirtioDevice

    // The backing server fd.
    Fd  int `json:"fd"`
}

func (device *VirtioConsoleDevice) dumpConsole(
    vchannel *VirtioChannel,
    fd int,
    read bool) error {

    // NOTE: This is just an example of how to
    // use a virtio device for the moment. This
    // will be done more rigorously shortly.
    for buf := range vchannel.incoming {
        if read {
            length, err := buf.Read(fd, 0, buf.Length())
            if err != nil {
                return err
            }
            if length == 0 {
                // Looks like the FD has been closed.
                // No problem, I suppose our job is done.
                // No more blocks will be read from this
                // channel.
                device.Debug("read done?")
                buf.length = 0
                vchannel.outgoing <- buf
                return nil

            } else {
                buf.length = length
            }

        } else {
            length, err := buf.Write(fd, 0, buf.Length())
            if err != nil {
                return err
            }
            if length == 0 {
                // Looks like the FD has been closed.
                // Okay, we'll just simulate it.
            } else {
                buf.length = length
            }
        }

        vchannel.outgoing <- buf
    }

    return nil
}

func (device *VirtioConsoleDevice) sendCtrl(
    port int,
    event int,
    value int) error {

    buf := <-device.Channels[2].incoming

    header := buf.Map(0, 8)

    if header.Size() < 8 {
        buf.length = 0
        device.Channels[2].outgoing <- buf
        return nil
    }

    header.Set32(0, uint32(port))
    header.Set16(4, uint16(event))
    header.Set16(6, uint16(value))
    buf.length = 8

    device.Channels[2].outgoing <- buf
    return nil
}

func (device *VirtioConsoleDevice) ctrlConsole(
    vchannel *VirtioChannel) error {

    for buf := range vchannel.incoming {

        header := buf.Map(0, 8)

        // Legit?
        if header.Size() < 8 {
            device.Debug("invalid ctrl packet?")
            vchannel.outgoing <- buf
            continue
        }

        id := header.Get32(0)
        event := header.Get16(4)
        value := header.Get16(6)

        // Return the buffer.
        vchannel.outgoing <- buf

        switch int(event) {
        case VirtioConsoleDeviceReady:
            vchannel.Debug("device-ready")
            device.sendCtrl(0, VirtioConsolePortAdd, 1)
            break

        case VirtioConsolePortAdd:
            vchannel.Debug("port-add?")
            break

        case VirtioConsolePortRemove:
            vchannel.Debug("port-remove?")
            break

        case VirtioConsolePortReady:
            vchannel.Debug("port-ready")

            if id == 0 && value == 1 {
                // Yes, this is a console.
                device.sendCtrl(0, VirtioConsolePortConsole, 1)
            }
            break

        case VirtioConsolePortConsole:
            vchannel.Debug("port-console?")
            break

        case VirtioConsolePortResize:
            vchannel.Debug("port-resize")
            break

        case VirtioConsolePortOpen:
            vchannel.Debug("port-open")
            break

        case VirtioConsolePortName:
            vchannel.Debug("port-name")
            break

        default:
            vchannel.Debug("unknown?")
            break
        }
    }

    return nil
}

func setupConsole(device *VirtioDevice) (Device, error) {

    device.SetFeatures(VirtioConsoleFMultiPort)

    device.Config.GrowTo(8)
    device.Config.Set32(4, 1)

    device.Channels[0] = device.NewVirtioChannel(128)
    device.Channels[1] = device.NewVirtioChannel(128)
    device.Channels[2] = device.NewVirtioChannel(32)
    device.Channels[3] = device.NewVirtioChannel(32)

    return &VirtioConsoleDevice{
        VirtioDevice: device}, nil
}

func NewVirtioMmioConsole(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeConsole)
    if err != nil {
        return nil, err
    }

    return setupConsole(device)
}

func NewVirtioPciConsole(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassMisc, VirtioTypeConsole)
    if err != nil {
        return nil, err
    }

    return setupConsole(device)
}

func (console *VirtioConsoleDevice) Attach(vm *platform.Vm, model *Model) error {
    err := console.VirtioDevice.Attach(vm, model)
    if err != nil {
        return err
    }

    // Make sure our FDs are not blocking.
    syscall.SetNonblock(0, false)
    syscall.SetNonblock(1, false)

    // Start our console process.
    go console.dumpConsole(console.Channels[0], 0, true)
    go console.dumpConsole(console.Channels[1], 1, false)
    go console.ctrlConsole(console.Channels[3])

    return nil
}
