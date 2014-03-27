package platform

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

        // Good to go.
        vcpus = append(vcpus, vcpu)
    }

    // We've okay.
    return vcpus, nil
}
