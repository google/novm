package control

import (
    "novmm/machine"
    "novmm/platform"
)

//
// State controls.
//

type StateRequest struct {
    // Include the model?
    Devices bool `json:"devices"`

    // Include the vcpus?
    Vcpus bool `json:"vcpus"`
}

type StateResult struct {
    // The model state.
    Devices []machine.DeviceInfo `json:"devices,omitempty"`

    // The vcpus state.
    Vcpus []platform.VcpuInfo `json:"vcpus,omitempty"`
}

func (control *Control) State(req *StateRequest, res *StateResult) error {

    // Include the model?
    if req.Devices {
        res.Devices = control.model.DeviceInfo()
    }

    // Include vcpus?
    if req.Vcpus {
        res.Vcpus = control.vm.VcpuInfo()
    }

    // That's it.
    // We let the serialization handle the rest.
    return nil
}
