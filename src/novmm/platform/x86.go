// +build i386 amd64
package platform

//
// x86 platform constants.
//
const (
    PageSize = 4096
)

//
// Our general purpose registers.
//
type Register int
type RegisterValue uint64

const (
    RAX Register = iota
    RBX
    RCX
    RDX
    RSI
    RDI
    RSP
    RBP
    R8
    R9
    R10
    R11
    R12
    R13
    R14
    R15
    RIP
    RFLAGS
)

//
// Special control registers.
//
type ControlRegister int
type ControlRegisterValue uint64

const (
    CR0 ControlRegister = iota
    CR2
    CR3
    CR4
    CR8
    EFER
    APIC_BASE
)

//
// Segment registers.
//
type Segment int
type SegmentValue struct {
    Base     uint64
    Limit    uint32
    Selector uint16
    Type     uint8
    Present  uint8
    Dpl      uint8
    Db       uint8
    L        uint8
    S        uint8
    G        uint8
    Avl      uint8
}

const (
    CS  Segment = iota
    DS
    ES
    FS
    GS
    SS
    TR
    LDT
)

//
// Segment descriptor registers.
//
type Descriptor int
type DescriptorValue struct {
    Base  uint64
    Limit uint16
}

const (
    GDT Descriptor = iota
    IDT
)

//
// Utility structure containing all registers.
//
type Registers struct {
    RAX    *RegisterValue
    RBX    *RegisterValue
    RCX    *RegisterValue
    RDX    *RegisterValue
    RSI    *RegisterValue
    RDI    *RegisterValue
    RSP    *RegisterValue
    RBP    *RegisterValue
    R8     *RegisterValue
    R9     *RegisterValue
    R10    *RegisterValue
    R11    *RegisterValue
    R12    *RegisterValue
    R13    *RegisterValue
    R14    *RegisterValue
    R15    *RegisterValue
    RIP    *RegisterValue
    RFLAGS *RegisterValue

    CR0       *ControlRegisterValue
    CR2       *ControlRegisterValue
    CR3       *ControlRegisterValue
    CR4       *ControlRegisterValue
    CR8       *ControlRegisterValue
    EFER      *ControlRegisterValue
    APIC_BASE *ControlRegisterValue

    CS  *SegmentValue
    DS  *SegmentValue
    ES  *SegmentValue
    FS  *SegmentValue
    GS  *SegmentValue
    SS  *SegmentValue
    TR  *SegmentValue
    LDT *SegmentValue

    IDT *DescriptorValue
    GDT *DescriptorValue
}

func (vcpu *Vcpu) getRegister(name string, reg Register) *RegisterValue {
    value, err := vcpu.GetRegister(reg)
    if err != nil {
        return nil
    }

    return &value
}

func (vcpu *Vcpu) getControlRegister(name string, reg ControlRegister) *ControlRegisterValue {
    value, err := vcpu.GetControlRegister(reg)
    if err != nil {
        return nil
    }

    return &value
}

func (vcpu *Vcpu) getSegment(name string, seg Segment) *SegmentValue {
    value, err := vcpu.GetSegment(seg)
    if err != nil {
        return nil
    }

    return &value
}

func (vcpu *Vcpu) getDescriptor(name string, desc Descriptor) *DescriptorValue {
    value, err := vcpu.GetDescriptor(desc)
    if err != nil {
        return nil
    }

    return &value
}

func (vcpu *Vcpu) GetRegisters() Registers {
    vcpu.Pause(false)
    defer vcpu.Unpause(false)

    var regs Registers

    regs.RAX = vcpu.getRegister("RAX", RAX)
    regs.RBX = vcpu.getRegister("RBX", RBX)
    regs.RCX = vcpu.getRegister("RCX", RCX)
    regs.RDX = vcpu.getRegister("RDX", RDX)
    regs.RSI = vcpu.getRegister("RSI", RSI)
    regs.RDI = vcpu.getRegister("RDI", RDI)
    regs.RSP = vcpu.getRegister("RSP", RSP)
    regs.RBP = vcpu.getRegister("RBP", RBP)
    regs.R8 = vcpu.getRegister("R8", R8)
    regs.R9 = vcpu.getRegister("R9", R9)
    regs.R10 = vcpu.getRegister("R10", R10)
    regs.R11 = vcpu.getRegister("R11", R11)
    regs.R12 = vcpu.getRegister("R12", R12)
    regs.R13 = vcpu.getRegister("R13", R13)
    regs.R14 = vcpu.getRegister("R14", R14)
    regs.R15 = vcpu.getRegister("R15", R15)
    regs.RIP = vcpu.getRegister("RIP", RIP)
    regs.RFLAGS = vcpu.getRegister("RFLAGS", RFLAGS)

    regs.CR0 = vcpu.getControlRegister("CR0", CR0)
    regs.CR2 = vcpu.getControlRegister("CR2", CR2)
    regs.CR3 = vcpu.getControlRegister("CR3", CR3)
    regs.CR4 = vcpu.getControlRegister("CR4", CR4)
    regs.CR8 = vcpu.getControlRegister("CR8", CR8)
    regs.EFER = vcpu.getControlRegister("EFER", EFER)
    regs.APIC_BASE = vcpu.getControlRegister("APIC_BASE", APIC_BASE)

    regs.CS = vcpu.getSegment("CS", CS)
    regs.DS = vcpu.getSegment("DS", DS)
    regs.ES = vcpu.getSegment("ES", ES)
    regs.FS = vcpu.getSegment("FS", FS)
    regs.GS = vcpu.getSegment("GS", GS)
    regs.SS = vcpu.getSegment("SS", SS)
    regs.TR = vcpu.getSegment("TR", TR)
    regs.LDT = vcpu.getSegment("LDT", LDT)

    regs.GDT = vcpu.getDescriptor("GDT", GDT)
    regs.IDT = vcpu.getDescriptor("IDT", IDT)

    return regs
}

func (vcpu *Vcpu) setRegister(name string, reg Register, value *RegisterValue) {
    if value != nil {
        vcpu.SetRegister(reg, *value)
    }
}

func (vcpu *Vcpu) setControlRegister(name string, reg ControlRegister, value *ControlRegisterValue) {
    if value != nil {
        vcpu.SetControlRegister(reg, *value, false)
    }
}

func (vcpu *Vcpu) setSegment(name string, seg Segment, value *SegmentValue) {
    if value != nil {
        vcpu.SetSegment(seg, *value, false)
    }
}

func (vcpu *Vcpu) setDescriptor(name string, desc Descriptor, value *DescriptorValue) {
    if value != nil {
        vcpu.SetDescriptor(desc, *value, false)
    }
}

func (vcpu *Vcpu) SetRegisters(regs Registers) {
    vcpu.Pause(false)
    defer vcpu.Unpause(false)

    vcpu.setRegister("RAX", RAX, regs.RAX)
    vcpu.setRegister("RBX", RBX, regs.RBX)
    vcpu.setRegister("RCX", RCX, regs.RCX)
    vcpu.setRegister("RDX", RDX, regs.RDX)
    vcpu.setRegister("RSI", RSI, regs.RSI)
    vcpu.setRegister("RDI", RDI, regs.RDI)
    vcpu.setRegister("RSP", RSP, regs.RSP)
    vcpu.setRegister("RBP", RBP, regs.RBP)
    vcpu.setRegister("R8", R8, regs.R8)
    vcpu.setRegister("R9", R9, regs.R9)
    vcpu.setRegister("R10", R10, regs.R10)
    vcpu.setRegister("R11", R11, regs.R11)
    vcpu.setRegister("R12", R12, regs.R12)
    vcpu.setRegister("R13", R13, regs.R13)
    vcpu.setRegister("R14", R14, regs.R14)
    vcpu.setRegister("R15", R15, regs.R15)
    vcpu.setRegister("RIP", RIP, regs.RIP)
    vcpu.setRegister("RFLAGS", RFLAGS, regs.RFLAGS)

    vcpu.setControlRegister("CR0", CR0, regs.CR0)
    vcpu.setControlRegister("CR2", CR2, regs.CR2)
    vcpu.setControlRegister("CR3", CR3, regs.CR3)
    vcpu.setControlRegister("CR4", CR4, regs.CR4)
    vcpu.setControlRegister("CR8", CR8, regs.CR8)
    vcpu.setControlRegister("EFER", EFER, regs.EFER)
    vcpu.setControlRegister("APIC_BASE", APIC_BASE, regs.APIC_BASE)

    vcpu.setSegment("CS", CS, regs.CS)
    vcpu.setSegment("DS", DS, regs.DS)
    vcpu.setSegment("ES", ES, regs.ES)
    vcpu.setSegment("FS", FS, regs.FS)
    vcpu.setSegment("GS", GS, regs.GS)
    vcpu.setSegment("SS", SS, regs.SS)
    vcpu.setSegment("TR", TR, regs.TR)
    vcpu.setSegment("LDT", LDT, regs.LDT)

    vcpu.setDescriptor("GDT", GDT, regs.GDT)
    vcpu.setDescriptor("IDT", IDT, regs.IDT)
}
