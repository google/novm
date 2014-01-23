package machine

import (
    "bytes"
    "encoding/json"
    "log"
)

type DeviceInfo struct {
    // Friendly name.
    Name string `json:"name"`

    // Driver name.
    Driver string `json:"driver"`

    // Device-specific info.
    Data interface{} `json:"data"`

    // Debugging?
    Debug bool `json:"debug"`
}

func (info DeviceInfo) Load() (Device, error) {

    // Find the appropriate driver.
    driver, ok := drivers[info.Driver]
    if !ok {
        return nil, DriverUnknown(info.Driver)
    }

    // Load the driver.
    device, err := driver(&info)
    if err != nil {
        return nil, err
    }

    if info.Data != nil {
        // Scratch data.
        buffer := bytes.NewBuffer(nil)

        // Encode the original object.
        json_encoder := json.NewEncoder(buffer)
        err = json_encoder.Encode(info.Data)
        if err != nil {
            return nil, err
        }

        // Decode a new object.
        // This will override all the default
        // settings in the initialized object.
        json_decoder := json.NewDecoder(buffer)
        log.Printf("Loading %s...", device.Name())
        err = json_decoder.Decode(device)
        if err != nil {
            return nil, err
        }
    }

    // Save the original device.
    // This will allow us to implement a
    // simple Save() method that serializes
    // the state of this device as it exists.
    info.Data = device

    // We're done.
    return device, nil
}
