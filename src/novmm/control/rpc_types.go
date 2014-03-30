package control

import (
    "novmm/loader"
    "novmm/machine"
    "novmm/platform"
)

//
// Rpc --
//
// This is basic state provided to the
// Rpc interface. All Rpc functions have
// access to this state (but nothing else).
//

type Rpc struct {
    // Our device model.
    model *machine.Model

    // Our underlying Vm object.
    vm  *platform.Vm

    // Our tracer.
    tracer *loader.Tracer
}

func NewRpc(
    model *machine.Model,
    vm *platform.Vm,
    tracer *loader.Tracer) *Rpc {

    return &Rpc{
        model:  model,
        vm:     vm,
        tracer: tracer,
    }
}

//
// The Noop --
//
// Many of our operations do not require
// a specific parameter or a specific return.
//
type Nop struct{}
