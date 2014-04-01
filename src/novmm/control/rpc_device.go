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

    // Drvier.
    Driver string `json:"driver"`

    // Debug?
    Debug bool `json:"debug"`

    // Pause?
    Paused bool `json:"paused"`
}

func (rpc *Rpc) Device(settings *DeviceSettings, nop *Nop) error {

    rn, err := regexp.Compile(settings.Name)
    if err != nil {
        return err
    }

    rd, err := regexp.Compile(settings.Driver)
    if err != nil {
        return err
    }

    for _, device := range rpc.model.Devices() {

        if rn.MatchString(device.Name()) &&
            rd.MatchString(device.Driver()) {

            device.SetDebugging(settings.Debug)

            if settings.Paused {
                err = device.Pause(true)
            } else {
                err = device.Unpause(true)
            }

            if err != nil {
                break
            }
        }
    }

    return err
}
