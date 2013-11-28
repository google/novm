package platform

import (
    "unsafe"
)

type Vm interface {
    NewVcpu() (Vcpu, error)
    Dispose() error
    Dump()

    Interrupt(irq Irq, level bool) error

    CreateUserMemory(size uint64) ([]byte, error)
    DeleteUserMemory(mmap []byte) error

    MapUserMemory(start Paddr, size uint64, mmap unsafe.Pointer) error
    UnmapUserMemory(start Paddr, size uint64) error

    MapReservedMemory(start Paddr, size uint64) error
    UnmapReservedMemory(start Paddr, size uint64) error

    MapSpecialMemory(start Paddr) (uint64, error)
    UnmapSpecialMemory(start Paddr) error
}

type Vcpu interface {
    Run(step bool) error
    Vm() Vm
    Dispose() error
    Dump()

    Translate(vaddr Vaddr) (Paddr, bool, bool, bool, error)

    SetRegister(reg Register, val RegisterValue) error
    GetRegister(reg Register) (RegisterValue, error)

    SetControlRegister(reg ControlRegister, val ControlRegisterValue, sync bool) error
    GetControlRegister(reg ControlRegister) (ControlRegisterValue, error)

    SetSegment(seg Segment, val SegmentValue, sync bool) error
    GetSegment(seg Segment) (SegmentValue, error)

    SetDescriptor(desc Descriptor, val DescriptorValue, sync bool) error
    GetDescriptor(desc Descriptor) (DescriptorValue, error)
}
