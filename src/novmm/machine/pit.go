package machine

import (
    "novmm/platform"
)

type Pit struct {
    BaseDevice

    // Our pit state.
    // Similar to the pit, we consider the platform
    // PIT to be an intrinsic part of our "pit".
    Pit platform.PitState `json:"pit"`
}

func NewPit(info *DeviceInfo) (Device, error) {
    pit := new(Pit)
    return pit, pit.init(info)
}

func (pit *Pit) Attach(vm *platform.Vm, model *Model) error {

    // Create our PIT.
    err := vm.CreatePit()
    if err != nil {
        return err
    }

    // Load state.
    err = vm.SetPit(pit.Pit)
    if err != nil {
        return err
    }

    // We're good.
    return nil
}

func (pit *Pit) Sync(vm *platform.Vm) error {

    var err error

    // Save our Pit state.
    pit.Pit, err = vm.GetPit()
    if err != nil {
        return err
    }

    // We're good.
    return nil
}
