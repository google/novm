package machine

import (
    "log"
    "novmm/platform"
)

type IoMap map[MemoryRegion]IoOperations
type IoHandlers map[MemoryRegion]*IoHandler

type BaseDevice struct {
    // Pointer to original device info.
    // This is reference in serialization.
    // (But is explicitly not exported, as
    // the device info will have a reference
    // back to this new device object).
    info *DeviceInfo
}

type Device interface {
    Name() string
    Driver() string

    PioHandlers() IoHandlers
    MmioHandlers() IoHandlers

    Attach(vm *platform.Vm, model *Model) error
    Sync(vm *platform.Vm) error

    Interrupt() error

    Debug(format string, v ...interface{})
    IsDebugging() bool
    SetDebugging(debug bool)
}

func (device *BaseDevice) init(info *DeviceInfo) error {
    // Save our original device info.
    // This isn't structural (hence no export).
    device.info = info
    return nil
}

func (device *BaseDevice) Name() string {
    return device.info.Name
}

func (device *BaseDevice) Driver() string {
    return device.info.Driver
}

func (device *BaseDevice) PioHandlers() IoHandlers {
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

func (device *BaseDevice) SetDebugging(debug bool) {
    device.info.Debug = debug
}

func (device *BaseDevice) Attach(vm *platform.Vm, model *Model) error {
    return nil
}

func (device *BaseDevice) Sync(vm *platform.Vm) error {
    return nil
}

func (device *BaseDevice) Interrupt() error {
    return nil
}
