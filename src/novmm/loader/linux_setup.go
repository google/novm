// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//+build i386 amd64
package loader

/*
#include <asm/types.h>
#include <linux/const.h>
#include <string.h>
#include <asm/bootparam.h>

// E820 codes.
const int E820Ram = E820_RAM;
const int E820Reserved = E820_RESERVED;
const int E820Acpi = E820_ACPI;

// Our E820 map.
static inline void e820_set_count(
    struct boot_params* boot,
    int count) {

    boot->e820_entries = count;
}
static inline void e820_set_region(
    struct boot_params* boot,
    int index,
    __u64 start,
    __u64 size,
    __u8 type) {

    boot->e820_map[index].addr = start;
    boot->e820_map[index].size = size;
    boot->e820_map[index].type = type;
}
static inline void set_header(
    struct boot_params* boot,
    __u64 initrd_addr,
    __u64 initrd_len,
    __u64 cmdline_addr) {

    boot->hdr.vid_mode = 0xffff;
    boot->hdr.type_of_loader = 0xff;
    boot->hdr.loadflags = 0;
    boot->hdr.setup_move_size = 0;
    boot->hdr.ramdisk_image = initrd_addr;
    boot->hdr.ramdisk_size = initrd_len;
    boot->hdr.heap_end_ptr = 0;
    boot->hdr.cmd_line_ptr = cmdline_addr;
    boot->hdr.setup_data = 0;
}
*/
import "C"

import (
	"novmm/machine"
	"novmm/platform"
	"unsafe"
)

func SetupLinuxBootParams(
	model *machine.Model,
	boot_params_data []byte,
	orig_boot_params_data []byte,
	cmdline_addr platform.Paddr,
	initrd_addr platform.Paddr,
	initrd_len uint64) error {

	// Grab a reference to our boot params struct.
	boot_params := (*C.struct_boot_params)(unsafe.Pointer(&boot_params_data[0]))

	// The setup header.
	// First step is to copy the existing setup_header
	// out of the given kernel image. We copy only the
	// header, and not the rest of the setup page.
	setup_start := 0x01f1
	setup_end := 0x0202 + int(orig_boot_params_data[0x0201])
	if setup_end > platform.PageSize {
		return InvalidSetupHeader
	}
	C.memcpy(
		unsafe.Pointer(&boot_params_data[setup_start]),
		unsafe.Pointer(&orig_boot_params_data[setup_start]),
		C.size_t(setup_end-setup_start))

	// Setup our BIOS memory map.
	// NOTE: We have to do this via C bindings. This is really
	// annoying, but basically because of the unaligned structures
	// in the struct_boot_params, the Go code generated here is
	// actually *incompatible* with the actual C layout.

	// First, the count.
	C.e820_set_count(boot_params, C.int(len(model.MemoryMap)))

	// Then, fill out the region information.
	for index, region := range model.MemoryMap {

		var memtype C.int
		switch region.MemoryType {
		case machine.MemoryTypeUser:
			memtype = C.E820Ram
		case machine.MemoryTypeReserved:
			memtype = C.E820Reserved
		case machine.MemoryTypeSpecial:
			memtype = C.E820Reserved
		case machine.MemoryTypeAcpi:
			memtype = C.E820Acpi
		}

		C.e820_set_region(
			boot_params,
			C.int(index),
			C.__u64(region.Start),
			C.__u64(region.Size),
			C.__u8(memtype))
	}

	// Set necessary setup header bits.
	C.set_header(
		boot_params,
		C.__u64(initrd_addr),
		C.__u64(initrd_len),
		C.__u64(cmdline_addr))

	// All done!
	return nil
}
