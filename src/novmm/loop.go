package main

import (
    "log"
    "novmm/loader"
    "novmm/machine"
    "novmm/platform"
)

func Loop(
    vcpu *platform.Vcpu,
    model *machine.Model,
    tracer *loader.Tracer) error {

    log.Print("Vcpu running...")

    for {
        // Enter the guest.
        err := vcpu.Run()

        // Trace if requested.
        trace_err := tracer.Trace(vcpu, vcpu.IsStepping())
        if trace_err != nil {
            return trace_err
        }

        // No reason for exit?
        if err == nil {
            return ExitWithoutReason
        }

        // Handle the error.
        switch err.(type) {
        case *platform.ExitPio:
            err = model.HandlePio(err.(*platform.ExitPio))

        case *platform.ExitMmio:
            err = model.HandleMmio(err.(*platform.ExitMmio))

        case *platform.ExitDebug:
            err = nil
        }

        // Error handling the exit.
        if err != nil {
            return err
        }
    }

    // Unreachable.
    return nil
}
