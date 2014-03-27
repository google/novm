package platform

type VcpuInfo struct {
    Regs  Registers  `json:"registers"`
    Cpuid []Cpuid    `json:"cpuid"`
    Msrs  []Msr      `json:"msrs"`
    LApic LApicState `json:"lapic"`
}

func (vm *Vm) LoadVcpus(spec []VcpuInfo) ([]*Vcpu, error) {

    vcpus := make([]*Vcpu, 0, 0)

    // Load all vcpus.
    for _, info := range spec {

        // Create a new vcpu.
        vcpu, err := vm.NewVcpu()
        if err != nil {
            return nil, err
        }

        // Ensure the registers are loaded.
        vcpu.SetRegisters(info.Regs)

        // Set our cpuid if we have one.
        if info.Cpuid != nil {
            err := vcpu.SetCpuid(info.Cpuid)
            if err != nil {
                return nil, err
            }
        }

        // Similarly, lapic if available.
        err = vcpu.SetLApic(info.LApic)
        if err != nil {
            return nil, err
        }

        // Finally, our MSRs.
        if info.Msrs != nil {
            err := vcpu.SetMsrs(info.Msrs)
            if err != nil {
                return nil, err
            }
        }

        // Good to go.
        vcpus = append(vcpus, vcpu)
    }

    // We've okay.
    return vcpus, nil
}

func NewVcpuInfo(vcpu *Vcpu) (VcpuInfo, error) {

    err := vcpu.Pause(false)
    if err != nil {
        return VcpuInfo{}, err
    }
    defer vcpu.Unpause(false)

    cpuid, err := vcpu.GetCpuid()
    if err != nil {
        return VcpuInfo{}, err
    }

    msrs, err := vcpu.GetMsrs()
    if err != nil {
        return VcpuInfo{}, err
    }

    lapic, err := vcpu.GetLApic()
    if err != nil {
        return VcpuInfo{}, err
    }

    return VcpuInfo{
        Regs:  vcpu.GetRegisters(),
        Cpuid: cpuid,
        Msrs:  msrs,
        LApic: lapic,
    }, nil
}
