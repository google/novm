package control

//
// High-level rpcs.
//

func (rpc *Rpc) Pause(nopin *Nop, nopout *Nop) error {
    return rpc.vm.Pause(true)
}

func (rpc *Rpc) Unpause(nopin *Nop, nopout *Nop) error {
    return rpc.vm.Unpause(true)
}
