package machine

import (
    "novmm/platform"
)

func (model *Model) LoadDevices(
    vm *platform.Vm,
    spec []DeviceInfo,
    debug bool) (Proxy, error) {

    // The first proxy decoded.
    var proxy Proxy

    // Load all devices.
    for _, info := range spec {
        device, err := info.Load()
        if err != nil {
            return nil, err
        }

        if debug {
            // Set our debug param.
            device.SetDebugging(debug)
        }

        // Try the attach.
        err = device.Attach(vm, model)
        if err != nil {
            return nil, err
        }

        // Add the device to our list.
        model.devices = append(model.devices, device)

        // Is this a proxy?
        if proxy == nil {
            proxy, _ = device.(Proxy)
        }
    }

    // Flush the model cache.
    return proxy, model.flush()
}
