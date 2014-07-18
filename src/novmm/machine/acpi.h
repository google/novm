/*
 * acpi.h
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

#include <linux/types.h>

long build_rsdp(void* start, __u32 rsdt_address, __u64 xsdt_address);

long build_rsdt(void* start, __u32 madt_address);
long build_xsdt(void* start, __u64 madt_address);

long build_dsdt(void* start);

long build_madt_device_lapic(void* start, __u8 processor_id, __u8 apic_id);
long build_madt_device_ioapic(void* start, __u8 ioapic_id, __u32 address, __u32 interrupt);
long build_madt(void* start, __u32 lapic_address, int vcpus, __u32 ioapic_address, __u32 ioapic_interrupt);
