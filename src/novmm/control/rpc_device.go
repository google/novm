package control

import (
    "regexp"
)

//
// Low-level device controls.
//

type DeviceSettings struct {
    // Name.
    Name string `json:"name"`

    // Debug?
    Debug bool `json:"debug"`
}

func (rpc *Rpc) Device(settings *DeviceSettings, nop *Nop) error {

    rp, err := regexp.Compile(settings.Name)
    if err != nil {
        return err
    }

    for _, device := range rpc.model.Devices() {
        if rp.MatchString(device.Name()) {
            device.SetDebugging(settings.Debug)
        }
    }

    return nil
}
