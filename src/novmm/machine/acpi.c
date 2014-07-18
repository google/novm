/*
 * apic.c
 *
 * Basic ACPI data structure generation.
 *
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "acpi.h"
#include <string.h>

static inline __u8 checksum(
    void* start,
    int length) {

    __u8 total = 0;
    __u8* val = (char*)start;

    while( length > 0 ) {
        total += *val;
        val++;
        length--;
    }

    return 0xff-total+1;
}

typedef struct rsdp {
    char signature[8];
    __u8 checksum;
    char oem_id[6];
    __u8 revision;
    __u32 rsdt_address;
    __u32 length;
    __u64 xsdt_address;
    char extended_checksum;
    char reserved[3];
} __attribute__((packed)) rsdp_t;

long build_rsdp(
    void* start,
    __u32 rsdt_address,
    __u64 xsdt_address) {

    rsdp_t *rsdp = (rsdp_t*)start;
    memcpy(rsdp->signature, "RSD PTR ", 8);
    memcpy(rsdp->oem_id, "PERVIR", 6);
    rsdp->revision = 2;
    rsdp->rsdt_address = rsdt_address;
    rsdp->length = sizeof(rsdp_t);
    rsdp->xsdt_address = xsdt_address;
    memset(rsdp->reserved, 0, 3);

    rsdp->checksum = checksum(start, 20);
    rsdp->extended_checksum = checksum(start, rsdp->length);

    return rsdp->length;
}

typedef struct acpi_header {
    char signature[4];
    __u32 length;
    __u8 revision;
    __u8 checksum;
    char oem_id[6];
    char oem_table_id[8];
    __u32 oem_revision;
    char asl_compiler_id[4];
    __u32 asl_compiler_rev;
} __attribute__((packed)) acpi_header_t;

typedef struct rsdt {
    acpi_header_t header;
    __u32 madt_address;
} __attribute__((packed)) rsdt_t;

long build_rsdt(
    void* start,
    __u32 madt_address)
{
    rsdt_t* rsdt = (rsdt_t*)start;

    memcpy(rsdt->header.signature, "RSDT", 4);
    rsdt->header.revision = 1;
    rsdt->header.length = sizeof(rsdt_t);
    memcpy(rsdt->header.oem_id, "PERVIR", 6);
    memcpy(rsdt->header.oem_table_id, "RSDT", 4);
    rsdt->header.oem_revision = 0;
    memcpy(rsdt->header.asl_compiler_id, "NOVM", 4);
    rsdt->header.asl_compiler_rev = 0;

    rsdt->madt_address = madt_address;
    rsdt->header.checksum = checksum(start, rsdt->header.length);

    return rsdt->header.length;
}

typedef struct xsdt {
    acpi_header_t header;
    __u64 madt_address;
} __attribute__((packed)) xsdt_t;

long build_xsdt(
    void* start,
    __u64 madt_address)
{
    xsdt_t* xsdt = (xsdt_t*)start;

    memcpy(xsdt->header.signature, "XSDT", 4);
    xsdt->header.revision = 1;
    xsdt->header.length = sizeof(xsdt_t);
    memcpy(xsdt->header.oem_id, "PERVIR", 6);
    memcpy(xsdt->header.oem_table_id, "XSDT", 4);
    xsdt->header.oem_revision = 0;
    memcpy(xsdt->header.asl_compiler_id, "NOVM", 4);
    xsdt->header.asl_compiler_rev = 0;

    xsdt->madt_address = madt_address;
    xsdt->header.checksum = checksum(start, xsdt->header.length);

    return xsdt->header.length;
}

typedef struct madt_device {
    __u8 type;
    __u8 length;
    char data[0];
} __attribute__((packed)) madt_device_t;

typedef struct madt_device_lapic {
    madt_device_t device;
    __u8 processor_id;
    __u8 apic_id;
    __u32 flags;
} __attribute__((packed)) madt_device_lapic_t;

long build_madt_device_lapic(
    void* start,
    __u8 processor_id,
    __u8 apic_id) {

    madt_device_lapic_t* lapic = (madt_device_lapic_t*)start;

    lapic->device.type = 0;
    lapic->device.length = sizeof(madt_device_lapic_t);
    lapic->processor_id = processor_id;
    lapic->apic_id = apic_id;
    lapic->flags = 0x1; /* Enabled. */

    return lapic->device.length;
}

typedef struct madt_device_ioapic {
    madt_device_t device;
    __u8 ioapic_id;
    __u8 reserved;
    __u32 address;
    __u32 interrupt;
} __attribute__((packed)) madt_device_ioapic_t;

long build_madt_device_ioapic(
    void* start,
    __u8 ioapic_id,
    __u32 address,
    __u32 interrupt) {

    madt_device_ioapic_t* ioapic = (madt_device_ioapic_t*)start;

    ioapic->device.type = 1;
    ioapic->device.length = sizeof(madt_device_ioapic_t);
    ioapic->ioapic_id = ioapic_id;
    ioapic->address = address;
    ioapic->interrupt = interrupt;

    return ioapic->device.length;
}

typedef struct dsdt {
    acpi_header_t header;
} dsdt_t;

long build_dsdt(
    void* start) {

    dsdt_t* dsdt = (dsdt_t*)start;

    memcpy(dsdt->header.signature, "DSDT", 4);
    dsdt->header.revision = 1;
    memcpy(dsdt->header.oem_id, "PERVIR", 6);
    memcpy(dsdt->header.oem_table_id, "DSDT", 4);
    dsdt->header.oem_revision = 0;
    memcpy(dsdt->header.asl_compiler_id, "NOVM", 4);
    dsdt->header.asl_compiler_rev = 0;

    dsdt->header.length = sizeof(dsdt_t);
    dsdt->header.checksum = checksum(start, dsdt->header.length);
    return dsdt->header.length;
}

typedef struct madt {
    acpi_header_t header;
    __u32 lapic_address;
    __u32 flags;
    madt_device_t devices[0];
} __attribute__((packed)) madt_t;

long build_madt(
    void* start,
    __u32 lapic_address,
    int vcpus,
    __u32 ioapic_address,
    __u32 ioapic_interrupt) {

    long offset = 0;
    int vcpu = 0;

    madt_t* madt = (madt_t*)start;

    memcpy(madt->header.signature, "ACPI", 4);
    madt->header.revision = 1;
    memcpy(madt->header.oem_id, "PERVIR", 6);
    memcpy(madt->header.oem_table_id, "MADT", 4);
    madt->header.oem_revision = 0;
    memcpy(madt->header.asl_compiler_id, "NOVM", 4);
    madt->header.asl_compiler_rev = 0;

    /* Build our local APIC entries. */
    for( vcpu = 0; vcpu < vcpus; vcpu += 1 ) {
        offset += build_madt_device_lapic(
            (void*)((char*)&madt->devices[0] + offset),
            vcpu, vcpu);
    }

    /* Build our I/O APIC entry. */
    offset += build_madt_device_ioapic(
        (void*)((char*)&madt->devices[0] + offset),
        0, ioapic_address, ioapic_interrupt);

    madt->header.length = sizeof(madt_t) + offset;
    madt->header.checksum = checksum(start, madt->header.length);
    return madt->header.length;
}
