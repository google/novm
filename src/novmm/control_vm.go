package main

type VmSettings struct {
}

func (control *Control) Vm(settings *VmSettings, ok *bool) error {
    *ok = true
    return nil
}
