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

func (model *Model) loadAcpi(vcpus uint) error {

    // Allocate our memory block.
    acpi_data, base_addr, err := model.Allocate(
        Acpi,
        "acpi",
        platform.Paddr(0xf0000),
        platform.PageSize,
        platform.Paddr(0xfe000),
        platform.PageSize)
    if err != nil {
        return err
    }

    // Load the MADT.
    madt_bytes := C.build_madt(
        unsafe.Pointer(&acpi_data[0]),
        C.__u32(0xfee00000), // Local APIC address.
        C.int(vcpus),
        C.__u32(0xfec00000), // I/O APIC address.
        C.__u32(0),          // I/O APIC interrupt.
    )
    log.Printf("acpi: MADT %x @ %x", madt_bytes, base_addr)

    // Align offset.
    offset := madt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the XSDT.
    xsdt_address := uint64(base_addr) + uint64(offset)
    xsdt_bytes := C.build_xsdt(
        unsafe.Pointer(&acpi_data[int(offset)]),
        C.__u64(base_addr), // MADT address.
    )
    log.Printf("acpi: XSDT %x @ %x", xsdt_bytes, xsdt_address)

    // Align offset.
    offset += xsdt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the RSDT.
    rsdt_address := uint64(base_addr) + uint64(offset)
    rsdt_bytes := C.build_rsdt(
        unsafe.Pointer(&acpi_data[int(offset)]),
        C.__u32(base_addr), // MADT address.
    )
    log.Printf("acpi: RSDT %x @ %x", rsdt_bytes, rsdt_address)

    // Align offset.
    offset += rsdt_bytes
    if offset%64 != 0 {
        offset += 64 - (offset % 64)
    }

    // Load the RSDP.
    rsdp_address := uint64(base_addr) + uint64(offset)
    rsdp_bytes := C.build_rsdp(
        unsafe.Pointer(&acpi_data[int(offset)]),
        C.__u32(rsdt_address), // RSDT address.
        C.__u64(xsdt_address), // XSDT address.
    )
    log.Printf("acpi: RSDP %x @ %x", rsdp_bytes, rsdp_address)

    // Everything went okay.
    return nil
}
