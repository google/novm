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
