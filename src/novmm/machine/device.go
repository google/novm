package machine

import (
    "log"
    "novmm/platform"
)

type IoMap map[MemoryRegion]IoOperations
type IoHandlers map[MemoryRegion]*IoHandler

type BaseDevice struct {
    // Pointer to original device info.
    info *DeviceInfo
}

type Device interface {
    Name() string

    PioHandlers() IoHandlers
    MmioHandlers() IoHandlers

    Attach(vm *platform.Vm, model *Model) error

    Interrupt() error

    Debug(format string, v ...interface{})
    IsDebugging() bool
}

func (device *BaseDevice) Init(info *DeviceInfo) error {
    // Save our original device info.
    // This isn't structural (hence the non-export).
    device.info = info
    return nil
}

func (device *BaseDevice) Name() string {
    return device.info.Name
}

func (device *BaseDevice) PioHandlers() IoHandlers {
    if device.Name() == "uart" {
        var d *BaseDevice
        d.info = nil
    }

    return IoHandlers{}
}

func (device *BaseDevice) MmioHandlers() IoHandlers {
    return IoHandlers{}
}

func (device *BaseDevice) Debug(format string, v ...interface{}) {
    if device.IsDebugging() {
        log.Printf(device.Name()+": "+format, v...)
    }
}

func (device *BaseDevice) IsDebugging() bool {
    return device.info.Debug
}

func (device *BaseDevice) Attach(vm *platform.Vm, model *Model) error {
    return nil
}

func (device *BaseDevice) Interrupt() error {
    return nil
}
