package machine

/*
#include "acpi.h"
*/
import "C"

import (
    "log"
    "novmm/platform"
    "unsafe"
)

type Acpi struct {
    BaseDevice

    Addr platform.Paddr `json:"address"`
    Data []byte         `json:"data"`
}

func NewAcpi(info *DeviceInfo) (Device, error) {
    acpi := new(Acpi)
    acpi.Addr = platform.Paddr(0xf0000)
    return acpi, acpi.Init(info)
}

func (acpi *Acpi) Attach(vm *platform.Vm, model *Model) error {

    // Do we already have data?
    rebuild := true
    if acpi.Data == nil {
        // Create our data.
        acpi.Data = make([]byte, platform.PageSize, platform.PageSize)
    } else {
        rebuild = false
    }

    // Allocate our memory block.
    err := model.Reserve(
        vm,
        acpi,
        MemoryTypeAcpi,
        acpi.Addr,
        platform.PageSize,
        acpi.Data)
    if err != nil {
        return err
    }

    // Already done.
    if !rebuild {
        return nil
    }

    // Load the MADT.
    madt_bytes := C.build_madt(
        unsafe.Pointer(&acpi.Data[0]),
        C.__u32(vm.LApic()),
        C.int(vm.VcpuCount()),
        C.__u32(vm.IOApic()),
        C.__u32(0), // I/O APIC interrupt?
    )
    log.Printf("acpi: MADT %x @ %x", madt_bytes, acpi.Addr)

    // Align offset.
    offset := madt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the DSDT.
    dsdt_address := uint64(acpi.Addr) + uint64(offset)
    dsdt_bytes := C.build_dsdt(
        unsafe.Pointer(&acpi.Data[int(offset)]),
    )
    log.Printf("acpi: DSDT %x @ %x", dsdt_bytes, dsdt_address)

    // Align offset.
    offset += dsdt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the XSDT.
    xsdt_address := uint64(acpi.Addr) + uint64(offset)
    xsdt_bytes := C.build_xsdt(
        unsafe.Pointer(&acpi.Data[int(offset)]),
        C.__u64(acpi.Addr), // MADT address.
    )
    log.Printf("acpi: XSDT %x @ %x", xsdt_bytes, xsdt_address)

    // Align offset.
    offset += xsdt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the RSDT.
    rsdt_address := uint64(acpi.Addr) + uint64(offset)
    rsdt_bytes := C.build_rsdt(
        unsafe.Pointer(&acpi.Data[int(offset)]),
        C.__u32(acpi.Addr), // MADT address.
    )
    log.Printf("acpi: RSDT %x @ %x", rsdt_bytes, rsdt_address)

    // Align offset.
    offset += rsdt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the RSDP.
    rsdp_address := uint64(acpi.Addr) + uint64(offset)
    rsdp_bytes := C.build_rsdp(
        unsafe.Pointer(&acpi.Data[int(offset)]),
        C.__u32(rsdt_address), // RSDT address.
        C.__u64(xsdt_address), // XSDT address.
    )
    log.Printf("acpi: RSDP %x @ %x", rsdp_bytes, rsdp_address)

    // Everything went okay.
    return nil
}
