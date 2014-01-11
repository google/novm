package platform

type VcpuInfo struct {
    Regs  []byte `json:"regs"`
    Sregs []byte `json:"sregs"`
}
