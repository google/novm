package platform

type VcpuInfo struct {
    Regs Registers `json:"registers"`
}

func NewVcpuInfo(vcpu *Vcpu) VcpuInfo {

    return VcpuInfo{
        Regs: vcpu.GetRegisters(),
    }
}
