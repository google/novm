package machine

import (
    "novmm/platform"
)

type Clock struct {
    BaseDevice

    // Our clock state.
    Clock platform.Clock `json:"clock"`
}

func NewClock(info *DeviceInfo) (Device, error) {
    clock := new(Clock)
    return clock, clock.init(info)
}

func (clock *Clock) Attach(vm *platform.Vm, model *Model) error {
    // We're good.
    return nil
}

func (clock *Clock) Save(vm *platform.Vm) error {

    var err error

    // Save our clock state.
    clock.Clock, err = vm.GetClock()
    if err != nil {
        return err
    }

    // We're good.
    return nil
}

func (clock *Clock) Load(vm *platform.Vm) error {
    // Load state.
    return vm.SetClock(clock.Clock)
}
