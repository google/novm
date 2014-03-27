package control

//
// Tracing & debug controls.
//

type TraceSettings struct {
    // Tracing?
    Enable bool `json:"enable"`
}

func (control *Control) Trace(settings *TraceSettings, nop *Nop) error {
    if settings.Enable {
        control.tracer.Enable()
    } else {
        control.tracer.Disable()
    }

    return nil
}
