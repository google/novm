package control

//
// State-related rpcs.
//

func (rpc *Rpc) State(nop *Nop, res *State) error {

    state, err := SaveState(rpc.vm, rpc.model)
    if err != nil {
        return err
    }

    // Save our state.
    res.Vcpus = state.Vcpus
    res.Devices = state.Devices

    return err
}

func (rpc *Rpc) Reload(in *Nop, out *Nop) error {

    // Pause the vm.
    // This is kept pausing for the entire reload().
    err := rpc.vm.Pause(false)
    if err != nil {
        return err
    }
    defer rpc.vm.Unpause(false)

    // Save a copy of the current state.
    state, err := SaveState(rpc.vm, rpc.model)
    if err != nil {
        return err
    }

    // Reload all vcpus.
    for i, vcpuspec := range state.Vcpus {
        err := rpc.vm.Vcpus()[i].Load(vcpuspec)
        if err != nil {
            return err
        }
    }

    // Reload all device state.
    return rpc.model.Load(rpc.vm)
}
