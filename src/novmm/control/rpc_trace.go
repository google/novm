package control

//
// Tracing & debug controls.
//

type TraceSettings struct {
    // Tracing?
    Enable bool `json:"enable"`
}

func (rpc *Rpc) Trace(settings *TraceSettings, nop *Nop) error {
    if settings.Enable {
        rpc.tracer.Enable()
    } else {
        rpc.tracer.Disable()
    }

    return nil
}
