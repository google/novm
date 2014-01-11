// +build linux i386 amd64
package platform

/*
#define type _type
#include <linux/kvm.h>
#include <string.h>

// VCPU Ioctls.
const int Run = KVM_RUN;
const int GetRegs = KVM_GET_REGS;
const int SetRegs = KVM_SET_REGS;
const int GetSRegs = KVM_GET_SREGS;
const int SetSRegs = KVM_SET_SREGS;

// VM Ioctls.
const int SetIdentityMapAddr = KVM_SET_IDENTITY_MAP_ADDR;
const int SetTssAddr = KVM_SET_TSS_ADDR;
*/
import "C"

import (
    "errors"
    "log"
    "syscall"
    "unsafe"
)

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

type Descriptor int
type DescriptorValue struct {
    Base  uint64
    Limit uint16
}

const (
    GDT Descriptor = iota
    IDT
)

const (
    PageSize = 4096
)

// Generic error for unknown registers.
var UnknownRegister = errors.New("Unknown Register")

func (vm *Vm) LApic() Paddr {
    // This is the default.
    return Paddr(0xfee00000)
}

func (vm *Vm) IOApic() Paddr {
    // This is the default.
    return Paddr(0xfec00000)
}

func (vcpu *Vcpu) Run(step bool) error {
    // Make sure our registers are flushed.
    err := vcpu.flushRegs()
    if err != nil {
        return err
    }
    err = vcpu.flushSRegs()
    if err != nil {
        return err
    }

    // Set single step if requested.
    if step {
        err = vcpu.setSingleStep(true)
        if err != nil {
            return err
        }
        defer vcpu.setSingleStep(false)
    }

    // Execute our run ioctl.
    for {
        _, _, e := syscall.Syscall(
            syscall.SYS_IOCTL,
            uintptr(vcpu.fd),
            uintptr(C.Run),
            0)

        if e == syscall.EINTR || e == syscall.EAGAIN {
            continue
        } else if e != 0 {
            return e
        } else {
            break
        }
    }

    return vcpu.GetExitError()
}

func (vcpu *Vcpu) refreshRegs(dirty bool) error {
    // Ensure that our registers are up-to-date.
    // NOTE: We don't use the sync registers capability
    // which will expose the registers via the shared page.
    // We don't really manipulate the registers often
    // beyond the bootloader, so there's really no sense
    // having the checks for dirty registers, etc.
    if !vcpu.regs_cached {
        _, _, e := syscall.Syscall(
            syscall.SYS_IOCTL,
            uintptr(vcpu.fd),
            uintptr(C.GetRegs),
            uintptr(unsafe.Pointer(&vcpu.regs)))
        if e != 0 {
            return e
        }
        vcpu.regs_cached = true
    }

    if dirty {
        vcpu.regs_dirty = true
    }
    return nil
}

func (vcpu *Vcpu) flushRegs() error {
    // Ensure that our registers are up-to-date.
    if vcpu.regs_dirty {
        _, _, e := syscall.Syscall(
            syscall.SYS_IOCTL,
            uintptr(vcpu.fd),
            uintptr(C.SetRegs),
            uintptr(unsafe.Pointer(&vcpu.regs)))
        if e != 0 {
            return e
        }
        vcpu.regs_dirty = false
    }

    vcpu.regs_cached = false
    return nil
}

func (vcpu *Vcpu) refreshSRegs(dirty bool) error {
    // Ensure that our registers are up-to-date.
    if !vcpu.sregs_cached {
        _, _, e := syscall.Syscall(
            syscall.SYS_IOCTL,
            uintptr(vcpu.fd),
            uintptr(C.GetSRegs),
            uintptr(unsafe.Pointer(&vcpu.sregs)))
        if e != 0 {
            return e
        }
        vcpu.sregs_cached = true
    }

    if dirty {
        vcpu.sregs_dirty = true
    }
    return nil
}

func (vcpu *Vcpu) flushSRegs() error {
    // Ensure that our registers are up-to-date.
    if vcpu.sregs_dirty {
        _, _, e := syscall.Syscall(
            syscall.SYS_IOCTL,
            uintptr(vcpu.fd),
            uintptr(C.SetSRegs),
            uintptr(unsafe.Pointer(&vcpu.sregs)))
        if e != 0 {
            return e
        }
        vcpu.sregs_dirty = false
    }

    vcpu.sregs_cached = false
    return nil
}

func (vcpu *Vcpu) SetRegister(reg Register, val RegisterValue) error {
    err := vcpu.refreshRegs(true)
    if err != nil {
        return err
    }

    switch reg {
    case RAX:
        vcpu.regs.rax = C.__u64(val)
    case RBX:
        vcpu.regs.rbx = C.__u64(val)
    case RCX:
        vcpu.regs.rcx = C.__u64(val)
    case RDX:
        vcpu.regs.rdx = C.__u64(val)
    case RSI:
        vcpu.regs.rsi = C.__u64(val)
    case RDI:
        vcpu.regs.rdi = C.__u64(val)
    case RSP:
        vcpu.regs.rsp = C.__u64(val)
    case RBP:
        vcpu.regs.rbp = C.__u64(val)
    case R8:
        vcpu.regs.r8 = C.__u64(val)
    case R9:
        vcpu.regs.r9 = C.__u64(val)
    case R10:
        vcpu.regs.r10 = C.__u64(val)
    case R11:
        vcpu.regs.r11 = C.__u64(val)
    case R12:
        vcpu.regs.r12 = C.__u64(val)
    case R13:
        vcpu.regs.r13 = C.__u64(val)
    case R14:
        vcpu.regs.r14 = C.__u64(val)
    case R15:
        vcpu.regs.r15 = C.__u64(val)
    case RIP:
        vcpu.regs.rip = C.__u64(val)
    case RFLAGS:
        vcpu.regs.rflags = C.__u64(val)
    default:
        return UnknownRegister
    }

    return nil
}

func (vcpu *Vcpu) GetRegister(reg Register) (RegisterValue, error) {
    err := vcpu.refreshRegs(false)
    if err != nil {
        return RegisterValue(0), err
    }

    switch reg {
    case RAX:
        return RegisterValue(vcpu.regs.rax), nil
    case RBX:
        return RegisterValue(vcpu.regs.rbx), nil
    case RCX:
        return RegisterValue(vcpu.regs.rcx), nil
    case RDX:
        return RegisterValue(vcpu.regs.rdx), nil
    case RSI:
        return RegisterValue(vcpu.regs.rsi), nil
    case RDI:
        return RegisterValue(vcpu.regs.rdi), nil
    case RSP:
        return RegisterValue(vcpu.regs.rsp), nil
    case RBP:
        return RegisterValue(vcpu.regs.rbp), nil
    case R8:
        return RegisterValue(vcpu.regs.r8), nil
    case R9:
        return RegisterValue(vcpu.regs.r9), nil
    case R10:
        return RegisterValue(vcpu.regs.r10), nil
    case R11:
        return RegisterValue(vcpu.regs.r11), nil
    case R12:
        return RegisterValue(vcpu.regs.r12), nil
    case R13:
        return RegisterValue(vcpu.regs.r13), nil
    case R14:
        return RegisterValue(vcpu.regs.r14), nil
    case R15:
        return RegisterValue(vcpu.regs.r15), nil
    case RIP:
        return RegisterValue(vcpu.regs.rip), nil
    case RFLAGS:
        return RegisterValue(vcpu.regs.rflags), nil
    }

    return RegisterValue(0), UnknownRegister
}

func (vcpu *Vcpu) dumpRegister(name string, reg Register) {
    value, err := vcpu.GetRegister(reg)

    if err != nil {
        log.Printf("  %s: error?", name, err)
    } else {
        log.Printf("  %s: 0x%08x", name, value)
    }
}

func (vcpu *Vcpu) dumpRegisters() {
    vcpu.dumpRegister("RAX", RAX)
    vcpu.dumpRegister("RBX", RBX)
    vcpu.dumpRegister("RCX", RCX)
    vcpu.dumpRegister("RDX", RDX)
    vcpu.dumpRegister("RSI", RSI)
    vcpu.dumpRegister("RDI", RDI)
    vcpu.dumpRegister("RSP", RSP)
    vcpu.dumpRegister("RBP", RBP)
    vcpu.dumpRegister("R8", R8)
    vcpu.dumpRegister("R9", R9)
    vcpu.dumpRegister("R10", R10)
    vcpu.dumpRegister("R11", R11)
    vcpu.dumpRegister("R12", R12)
    vcpu.dumpRegister("R13", R13)
    vcpu.dumpRegister("R14", R14)
    vcpu.dumpRegister("R15", R15)
    vcpu.dumpRegister("RIP", RIP)
    vcpu.dumpRegister("RFLAGS", RFLAGS)
}

func (vcpu *Vcpu) SetControlRegister(
    reg ControlRegister,
    val ControlRegisterValue,
    sync bool) error {

    err := vcpu.refreshSRegs(true)
    if err != nil {
        return err
    }

    switch reg {
    case CR0:
        vcpu.sregs.cr0 = C.__u64(val)
    case CR2:
        vcpu.sregs.cr2 = C.__u64(val)
    case CR3:
        vcpu.sregs.cr3 = C.__u64(val)
    case CR4:
        vcpu.sregs.cr4 = C.__u64(val)
    case CR8:
        vcpu.sregs.cr8 = C.__u64(val)
    case EFER:
        vcpu.sregs.efer = C.__u64(val)
    case APIC_BASE:
        vcpu.sregs.apic_base = C.__u64(val)
    default:
        return UnknownRegister
    }

    if sync {
        err = vcpu.flushSRegs()
        if err != nil {
            return err
        }
    }

    return nil
}

func (vcpu *Vcpu) GetControlRegister(reg ControlRegister) (ControlRegisterValue, error) {
    err := vcpu.refreshSRegs(false)
    if err != nil {
        return ControlRegisterValue(0), err
    }

    switch reg {
    case CR0:
        return ControlRegisterValue(vcpu.sregs.cr0), nil
    case CR2:
        return ControlRegisterValue(vcpu.sregs.cr2), nil
    case CR3:
        return ControlRegisterValue(vcpu.sregs.cr3), nil
    case CR4:
        return ControlRegisterValue(vcpu.sregs.cr4), nil
    case CR8:
        return ControlRegisterValue(vcpu.sregs.cr8), nil
    case EFER:
        return ControlRegisterValue(vcpu.sregs.efer), nil
    case APIC_BASE:
        return ControlRegisterValue(vcpu.sregs.apic_base), nil
    }

    return ControlRegisterValue(0), UnknownRegister
}

func (vcpu *Vcpu) dumpControlRegister(name string, reg ControlRegister) {
    value, err := vcpu.GetControlRegister(reg)

    if err != nil {
        log.Printf("  %s: error?", name, err)
    } else {
        log.Printf("  %s: 0x%08x", name, value)
    }
}

func (vcpu *Vcpu) dumpControlRegisters() {
    vcpu.dumpControlRegister("CR0", CR0)
    vcpu.dumpControlRegister("CR2", CR2)
    vcpu.dumpControlRegister("CR3", CR3)
    vcpu.dumpControlRegister("CR4", CR4)
    vcpu.dumpControlRegister("CR8", CR8)
    vcpu.dumpControlRegister("EFER", EFER)
    vcpu.dumpControlRegister("APIC_BASE", APIC_BASE)
}

func (vcpu *Vcpu) SetSegment(
    seg Segment,
    val SegmentValue,
    sync bool) error {

    err := vcpu.refreshSRegs(true)
    if err != nil {
        return err
    }

    switch seg {
    case CS:
        vcpu.sregs.cs.base = C.__u64(val.Base)
        vcpu.sregs.cs.limit = C.__u32(val.Limit)
        vcpu.sregs.cs.selector = C.__u16(val.Selector)
        vcpu.sregs.cs._type = C.__u8(val.Type)
        vcpu.sregs.cs.present = C.__u8(val.Present)
        vcpu.sregs.cs.dpl = C.__u8(val.Dpl)
        vcpu.sregs.cs.db = C.__u8(val.Db)
        vcpu.sregs.cs.s = C.__u8(val.S)
        vcpu.sregs.cs.l = C.__u8(val.L)
        vcpu.sregs.cs.g = C.__u8(val.G)
        vcpu.sregs.cs.avl = C.__u8(val.Avl)
    case DS:
        vcpu.sregs.ds.base = C.__u64(val.Base)
        vcpu.sregs.ds.limit = C.__u32(val.Limit)
        vcpu.sregs.ds.selector = C.__u16(val.Selector)
        vcpu.sregs.ds._type = C.__u8(val.Type)
        vcpu.sregs.ds.present = C.__u8(val.Present)
        vcpu.sregs.ds.dpl = C.__u8(val.Dpl)
        vcpu.sregs.ds.db = C.__u8(val.Db)
        vcpu.sregs.ds.s = C.__u8(val.S)
        vcpu.sregs.ds.l = C.__u8(val.L)
        vcpu.sregs.ds.g = C.__u8(val.G)
        vcpu.sregs.ds.avl = C.__u8(val.Avl)
    case ES:
        vcpu.sregs.es.base = C.__u64(val.Base)
        vcpu.sregs.es.limit = C.__u32(val.Limit)
        vcpu.sregs.es.selector = C.__u16(val.Selector)
        vcpu.sregs.es._type = C.__u8(val.Type)
        vcpu.sregs.es.present = C.__u8(val.Present)
        vcpu.sregs.es.dpl = C.__u8(val.Dpl)
        vcpu.sregs.es.db = C.__u8(val.Db)
        vcpu.sregs.es.s = C.__u8(val.S)
        vcpu.sregs.es.l = C.__u8(val.L)
        vcpu.sregs.es.g = C.__u8(val.G)
        vcpu.sregs.es.avl = C.__u8(val.Avl)
    case FS:
        vcpu.sregs.fs.base = C.__u64(val.Base)
        vcpu.sregs.fs.limit = C.__u32(val.Limit)
        vcpu.sregs.fs.selector = C.__u16(val.Selector)
        vcpu.sregs.fs._type = C.__u8(val.Type)
        vcpu.sregs.fs.present = C.__u8(val.Present)
        vcpu.sregs.fs.dpl = C.__u8(val.Dpl)
        vcpu.sregs.fs.db = C.__u8(val.Db)
        vcpu.sregs.fs.s = C.__u8(val.S)
        vcpu.sregs.fs.l = C.__u8(val.L)
        vcpu.sregs.fs.g = C.__u8(val.G)
        vcpu.sregs.fs.avl = C.__u8(val.Avl)
    case GS:
        vcpu.sregs.gs.base = C.__u64(val.Base)
        vcpu.sregs.gs.limit = C.__u32(val.Limit)
        vcpu.sregs.gs.selector = C.__u16(val.Selector)
        vcpu.sregs.gs._type = C.__u8(val.Type)
        vcpu.sregs.gs.present = C.__u8(val.Present)
        vcpu.sregs.gs.dpl = C.__u8(val.Dpl)
        vcpu.sregs.gs.db = C.__u8(val.Db)
        vcpu.sregs.gs.s = C.__u8(val.S)
        vcpu.sregs.gs.l = C.__u8(val.L)
        vcpu.sregs.gs.g = C.__u8(val.G)
        vcpu.sregs.gs.avl = C.__u8(val.Avl)
    case SS:
        vcpu.sregs.ss.base = C.__u64(val.Base)
        vcpu.sregs.ss.limit = C.__u32(val.Limit)
        vcpu.sregs.ss.selector = C.__u16(val.Selector)
        vcpu.sregs.ss._type = C.__u8(val.Type)
        vcpu.sregs.ss.present = C.__u8(val.Present)
        vcpu.sregs.ss.dpl = C.__u8(val.Dpl)
        vcpu.sregs.ss.db = C.__u8(val.Db)
        vcpu.sregs.ss.s = C.__u8(val.S)
        vcpu.sregs.ss.l = C.__u8(val.L)
        vcpu.sregs.ss.g = C.__u8(val.G)
        vcpu.sregs.ss.avl = C.__u8(val.Avl)
    case TR:
        vcpu.sregs.tr.base = C.__u64(val.Base)
        vcpu.sregs.tr.limit = C.__u32(val.Limit)
        vcpu.sregs.tr.selector = C.__u16(val.Selector)
        vcpu.sregs.tr._type = C.__u8(val.Type)
        vcpu.sregs.tr.present = C.__u8(val.Present)
        vcpu.sregs.tr.dpl = C.__u8(val.Dpl)
        vcpu.sregs.tr.db = C.__u8(val.Db)
        vcpu.sregs.tr.s = C.__u8(val.S)
        vcpu.sregs.tr.l = C.__u8(val.L)
        vcpu.sregs.tr.g = C.__u8(val.G)
        vcpu.sregs.tr.avl = C.__u8(val.Avl)
    case LDT:
        vcpu.sregs.ldt.base = C.__u64(val.Base)
        vcpu.sregs.ldt.limit = C.__u32(val.Limit)
        vcpu.sregs.ldt.selector = C.__u16(val.Selector)
        vcpu.sregs.ldt._type = C.__u8(val.Type)
        vcpu.sregs.ldt.present = C.__u8(val.Present)
        vcpu.sregs.ldt.dpl = C.__u8(val.Dpl)
        vcpu.sregs.ldt.db = C.__u8(val.Db)
        vcpu.sregs.ldt.s = C.__u8(val.S)
        vcpu.sregs.ldt.l = C.__u8(val.L)
        vcpu.sregs.ldt.g = C.__u8(val.G)
        vcpu.sregs.ldt.avl = C.__u8(val.Avl)
    default:
        return UnknownRegister
    }

    if sync {
        err = vcpu.flushSRegs()
        if err != nil {
            return err
        }
    }

    return nil
}

func (vcpu *Vcpu) GetSegment(seg Segment) (SegmentValue, error) {
    err := vcpu.refreshSRegs(false)
    if err != nil {
        return SegmentValue{}, err
    }

    switch seg {
    case CS:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.cs.base),
                Limit:    uint32(vcpu.sregs.cs.limit),
                Selector: uint16(vcpu.sregs.cs.selector),
                Type:     uint8(vcpu.sregs.cs._type),
                Present:  uint8(vcpu.sregs.cs.present),
                Dpl:      uint8(vcpu.sregs.cs.dpl),
                Db:       uint8(vcpu.sregs.cs.db),
                S:        uint8(vcpu.sregs.cs.s),
                L:        uint8(vcpu.sregs.cs.l),
                G:        uint8(vcpu.sregs.cs.g),
                Avl:      uint8(vcpu.sregs.cs.avl)},
            nil
    case DS:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.ds.base),
                Limit:    uint32(vcpu.sregs.ds.limit),
                Selector: uint16(vcpu.sregs.ds.selector),
                Type:     uint8(vcpu.sregs.ds._type),
                Present:  uint8(vcpu.sregs.ds.present),
                Dpl:      uint8(vcpu.sregs.ds.dpl),
                Db:       uint8(vcpu.sregs.ds.db),
                S:        uint8(vcpu.sregs.ds.s),
                L:        uint8(vcpu.sregs.ds.l),
                G:        uint8(vcpu.sregs.ds.g),
                Avl:      uint8(vcpu.sregs.ds.avl)},
            nil
    case ES:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.es.base),
                Limit:    uint32(vcpu.sregs.es.limit),
                Selector: uint16(vcpu.sregs.es.selector),
                Type:     uint8(vcpu.sregs.es._type),
                Present:  uint8(vcpu.sregs.es.present),
                Dpl:      uint8(vcpu.sregs.es.dpl),
                Db:       uint8(vcpu.sregs.es.db),
                S:        uint8(vcpu.sregs.es.s),
                L:        uint8(vcpu.sregs.es.l),
                G:        uint8(vcpu.sregs.es.g),
                Avl:      uint8(vcpu.sregs.es.avl)},
            nil
    case FS:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.fs.base),
                Limit:    uint32(vcpu.sregs.fs.limit),
                Selector: uint16(vcpu.sregs.fs.selector),
                Type:     uint8(vcpu.sregs.fs._type),
                Present:  uint8(vcpu.sregs.fs.present),
                Dpl:      uint8(vcpu.sregs.fs.dpl),
                Db:       uint8(vcpu.sregs.fs.db),
                S:        uint8(vcpu.sregs.fs.s),
                L:        uint8(vcpu.sregs.fs.l),
                G:        uint8(vcpu.sregs.fs.g),
                Avl:      uint8(vcpu.sregs.fs.avl)},
            nil
    case GS:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.gs.base),
                Limit:    uint32(vcpu.sregs.gs.limit),
                Selector: uint16(vcpu.sregs.gs.selector),
                Type:     uint8(vcpu.sregs.gs._type),
                Present:  uint8(vcpu.sregs.gs.present),
                Dpl:      uint8(vcpu.sregs.gs.dpl),
                Db:       uint8(vcpu.sregs.gs.db),
                S:        uint8(vcpu.sregs.gs.s),
                L:        uint8(vcpu.sregs.gs.l),
                G:        uint8(vcpu.sregs.gs.g),
                Avl:      uint8(vcpu.sregs.gs.avl)},
            nil
    case SS:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.ss.base),
                Limit:    uint32(vcpu.sregs.ss.limit),
                Selector: uint16(vcpu.sregs.ss.selector),
                Type:     uint8(vcpu.sregs.ss._type),
                Present:  uint8(vcpu.sregs.ss.present),
                Dpl:      uint8(vcpu.sregs.ss.dpl),
                Db:       uint8(vcpu.sregs.ss.db),
                S:        uint8(vcpu.sregs.ss.s),
                L:        uint8(vcpu.sregs.ss.l),
                G:        uint8(vcpu.sregs.ss.g),
                Avl:      uint8(vcpu.sregs.ss.avl)},
            nil
    case TR:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.tr.base),
                Limit:    uint32(vcpu.sregs.tr.limit),
                Selector: uint16(vcpu.sregs.tr.selector),
                Type:     uint8(vcpu.sregs.tr._type),
                Present:  uint8(vcpu.sregs.tr.present),
                Dpl:      uint8(vcpu.sregs.tr.dpl),
                Db:       uint8(vcpu.sregs.tr.db),
                S:        uint8(vcpu.sregs.tr.s),
                L:        uint8(vcpu.sregs.tr.l),
                G:        uint8(vcpu.sregs.tr.g),
                Avl:      uint8(vcpu.sregs.tr.avl)},
            nil
    case LDT:
        return SegmentValue{
                Base:     uint64(vcpu.sregs.ldt.base),
                Limit:    uint32(vcpu.sregs.ldt.limit),
                Selector: uint16(vcpu.sregs.ldt.selector),
                Type:     uint8(vcpu.sregs.ldt._type),
                Present:  uint8(vcpu.sregs.ldt.present),
                Dpl:      uint8(vcpu.sregs.ldt.dpl),
                Db:       uint8(vcpu.sregs.ldt.db),
                S:        uint8(vcpu.sregs.ldt.s),
                L:        uint8(vcpu.sregs.ldt.l),
                G:        uint8(vcpu.sregs.ldt.g),
                Avl:      uint8(vcpu.sregs.ldt.avl)},
            nil
    }

    return SegmentValue{}, UnknownRegister
}

func (vcpu *Vcpu) dumpSegment(name string, seg Segment) {
    log.Printf("  segment %s:", name)
    value, err := vcpu.GetSegment(seg)

    if err != nil {
        log.Print("    error?", err)
        return
    }

    log.Printf("    base: 0x%08x", value.Base)
    log.Printf("    limit: 0x%08x", value.Limit)
    log.Print("    selector: ", value.Selector)
    log.Printf("    type: %x", value.Type)
    log.Print("    present: ", value.Present)
    log.Printf("    dpl: %x", value.Dpl)
    log.Print("    s: ", value.S)
    log.Print("    l: ", value.L)
    log.Print("    g: ", value.G)
    log.Print("    d: ", value.Db)
    log.Print("    avl: ", value.Avl)
}

func (vcpu *Vcpu) dumpSegments() {
    vcpu.dumpSegment("CS", CS)
    vcpu.dumpSegment("DS", DS)
    vcpu.dumpSegment("ES", ES)
    vcpu.dumpSegment("FS", FS)
    vcpu.dumpSegment("GS", GS)
    vcpu.dumpSegment("SS", SS)
    vcpu.dumpSegment("TR", TR)
    vcpu.dumpSegment("LDT", LDT)
}

func (vcpu *Vcpu) SetDescriptor(
    desc Descriptor,
    val DescriptorValue,
    sync bool) error {

    err := vcpu.refreshSRegs(true)
    if err != nil {
        return err
    }

    switch desc {
    case GDT:
        vcpu.sregs.gdt.base = C.__u64(val.Base)
        vcpu.sregs.gdt.limit = C.__u16(val.Limit)
    case IDT:
        vcpu.sregs.idt.base = C.__u64(val.Base)
        vcpu.sregs.idt.limit = C.__u16(val.Limit)
    default:
        return UnknownRegister
    }

    if sync {
        err = vcpu.flushSRegs()
        if err != nil {
            return err
        }
    }

    return nil
}

func (vcpu *Vcpu) GetDescriptor(desc Descriptor) (DescriptorValue, error) {
    err := vcpu.refreshSRegs(false)
    if err != nil {
        return DescriptorValue{}, err
    }

    switch desc {
    case GDT:
        return DescriptorValue{
                Base:  uint64(vcpu.sregs.gdt.base),
                Limit: uint16(vcpu.sregs.gdt.limit)},
            nil
    case IDT:
        return DescriptorValue{
                Base:  uint64(vcpu.sregs.idt.base),
                Limit: uint16(vcpu.sregs.idt.limit)},
            nil
    }

    return DescriptorValue{}, UnknownRegister
}

func (vcpu *Vcpu) dumpDescriptor(name string, desc Descriptor) {
    log.Printf("  descriptor %s:", name)
    value, err := vcpu.GetDescriptor(desc)

    if err != nil {
        log.Print("    error?", err)
        return
    }

    log.Printf("    base: 0x%08x", value.Base)
    log.Printf("    limit: 0x%08x", value.Limit)
}

func (vcpu *Vcpu) dumpDescriptors() {
    vcpu.dumpDescriptor("GDT", GDT)
    vcpu.dumpDescriptor("IDT", IDT)
}

func (vcpu *Vcpu) Dump() {
    log.Printf("Vcpu (id: %d):", vcpu.vcpu_id)

    vcpu.dumpRegisters()
    vcpu.dumpControlRegisters()
    vcpu.dumpSegments()
    vcpu.dumpDescriptors()
}

func (vm *Vm) SizeSpecialMemory() uint64 {
    return 4 * PageSize
}

func (vm *Vm) MapSpecialMemory(addr Paddr) error {

    // We require 1 page for the identity map.
    err := vm.MapReservedMemory(addr, PageSize)
    if err != nil {
        return err
    }

    // Set the EPT identity map.
    // (This requires a single page).
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.SetIdentityMapAddr),
        uintptr(unsafe.Pointer(&addr)))
    if e != 0 {
        log.Printf("Unable to set identity map to 0x%08x!", addr)
        return e
    }

    // We require 3 pages for the TSS address.
    err = vm.MapReservedMemory(addr+PageSize, 3*PageSize)
    if err != nil {
        return err
    }

    // Set the TSS address to above.
    // (This requires three pages).
    _, _, e = syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.SetTssAddr),
        uintptr(addr+PageSize))
    if e != 0 {
        log.Printf("Unable to set TSS ADDR to 0x%08x!", addr+PageSize)
        return e
    }

    // We're okay.
    return nil
}
