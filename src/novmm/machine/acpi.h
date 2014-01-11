/*
 * acpi.h
 *
 * Basic ACPI data structure generation.
 */

#include <linux/types.h>

long build_rsdp(void* start, __u32 rsdt_address, __u64 xsdt_address);
long build_rsdt(void* start, __u32 dsdt_address, __u32 madt_address);
long build_xsdt(void* start, __u64 dsdt_address, __u64 madt_address);

long build_dsdt(void* start);
long build_madt_device_lapic(void* start, __u8 processor_id, __u8 apic_id);
long build_madt_device_ioapic(void* start, __u8 ioapic_id, __u32 address, __u32 interrupt);
long build_madt(void* start, __u32 lapic_address, int vcpus, __u32 ioapic_address, __u32 ioapic_interrupt);
