package machine

import (
    "bytes"
    "encoding/json"
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

func (info *DeviceInfo) Load(data interface{}) error {

    // Scratch data.
    buffer := bytes.NewBuffer(nil)

    // Encode the original object.
    json_encoder := json.NewEncoder(buffer)
    err := json_encoder.Encode(info.Data)
    if err != nil {
        return err
    }

    // Decode a new object.
    json_decoder := json.NewDecoder(buffer)
    err = json_decoder.Decode(data)
    if err != nil {
        return err
    }

    // Save the result.
    info.Data = data

    // We're done.
    return nil
}
