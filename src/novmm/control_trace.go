package main

type TraceSettings struct {
    // Tracing?
    Enable bool `json:"enable"`
}

func (control *Control) Trace(settings *TraceSettings, ok *bool) error {
    if settings.Enable {
        control.tracer.Enable()
    } else {
        control.tracer.Disable()
    }
    *ok = true
    return nil
}
