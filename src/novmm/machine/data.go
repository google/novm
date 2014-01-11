package machine

import (
    "bytes"
    "encoding/json"
    "log"
    "novmm/platform"
)

func (model *Model) LoadDevices(vm *platform.Vm, data []byte) error {

    // Decode a new object.
    json_decoder := json.NewDecoder(bytes.NewBuffer([]byte(data)))
    spec := make([]*DeviceInfo, 0, 0)
    err := json_decoder.Decode(&spec)
    if err != nil {
        return err
    }

    // Load all devices.
    for _, info := range spec {
        device, err := info.Load()
        if err != nil {
            return err
        }

        // Good old context.
        log.Printf("model: attaching %s...", device.Name())

        err = device.Attach(vm, model)
        if err != nil {
            return err
        }

        // Add the device to our list.
        model.devices = append(model.devices, device)
    }

    // Flush the model cache.
    return model.flush()

    // We've okay.
    return nil
}
