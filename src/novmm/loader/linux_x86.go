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

// Our expected segment selectors.
#define __BOOT_CS 2
#define __BOOT_DS 3
#define __BOOT_TR1 4
#define __BOOT_TR2 5
const int BootCsSelector = __BOOT_CS * sizeof(__u64);
const int BootDsSelector = __BOOT_DS * sizeof(__u64);
const int BootTrSelector = __BOOT_TR1 * sizeof(__u64);

#define GDT_ENTRY(flags, base, limit)               \
    ((((base)  & _AC(0xff000000,ULL)) << (56-24)) | \
     (((flags) & _AC(0x0000f0ff,ULL)) << 40) |      \
     (((limit) & _AC(0x000f0000,ULL)) << (48-16)) | \
     (((base)  & _AC(0x00ffffff,ULL)) << 16) |      \
     (((limit) & _AC(0x0000ffff,ULL))))

// Our boot GDT table.
static inline void build_32bit_gdt(void* page) {
    __u64* entry = (__u64*)page;
    entry[__BOOT_CS] = GDT_ENTRY(0xc09a, 0, 0xfffff);
    entry[__BOOT_DS] = GDT_ENTRY(0xc092, 0, 0xfffff);
    entry[__BOOT_TR1] = GDT_ENTRY(0x8089, 0, 0x0);
    entry[__BOOT_TR2] = GDT_ENTRY(0x0000, 0, 0x0);
}
static inline void build_64bit_gdt(void* page) {
    __u64* entry = (__u64*)page;
    entry[__BOOT_CS] = GDT_ENTRY(0xa09a, 0, 0xfffff);
    entry[__BOOT_DS] = GDT_ENTRY(0xc092, 0, 0xfffff);
    entry[__BOOT_TR1] = GDT_ENTRY(0x808b, 0, 0xfffff);
    entry[__BOOT_TR2] = GDT_ENTRY(0x0000, 0, 0);
}

// Our page table builders.
static inline void build_pml4(
    void* page,
    __u64 pgd_addr,
    int size) {

    // We will only build a single entry.
    // Thus, we will only be able to address 512GB from within
    // the linux startup routines. I think we'll be okay.
    __u64* entry = (__u64*)page;

    // Encodes: present, writable
    *entry = (pgd_addr & 0x000ffffffffff000) | 0x3;
}
static inline void build_pgd(
    void* page,
    __u64 pde_addr,
    int size) {

    // Wait! Here we only build a single entry.
    // Uh-oh, that means our startup routine is now further
    // limited -- to only 1GB! We'll probably still be okay.
    __u64* entry = (__u64*)page;

    // Encodes: present, writable
    *entry = (pde_addr & 0x000ffffffffff000) | 0x3;
}
static inline void build_pde(
    void* page,
    int size) {

    // Here we build an entry for every possible
    // index. Each entry represents 2MB addressable.
    __u64* entry = (__u64*)page;
    int i = 0, max = size / sizeof(*entry);
    for (i = 0; i < max; i += 1) {
        // Encodes: PSE (2mb), present, writable
        entry[i] = ((((__u64)i) << 21) & 0x000fffffffe00000) | 0x83;
    }
}
*/
import "C"

import (
	"log"
	"novmm/machine"
	"novmm/platform"
	"unsafe"
)

var Linux32Convention = Convention{
	instruction: platform.RIP,
	arguments: []platform.Register{
		platform.RCX,
		platform.RDX,
	},
	rvalue: platform.RAX,
	stack:  platform.RSI,
}

var Linux64Convention = Convention{
	instruction: platform.RIP,
	arguments: []platform.Register{
		platform.RDI,
		platform.RSI,
		platform.RDX,
		platform.RCX,
	},
	rvalue: platform.RAX,
	stack:  platform.RBP,
}

func SetupLinux(
	vcpu *platform.Vcpu,
	model *machine.Model,
	orig_boot_data []byte,
	entry_point uint64,
	is_64bit bool,
	initrd_addr platform.Paddr,
	initrd_len uint64,
	cmdline_addr platform.Paddr) error {

	// Copy in the GDT table.
	// These match the segments below.
	gdt_addr, gdt, err := model.Allocate(
		machine.MemoryTypeUser,
		0,                 // Start.
		model.Max(),       // End.
		platform.PageSize, // Size.
		false)             // From bottom.
	if err != nil {
		return err
	}
	if is_64bit {
		C.build_64bit_gdt(unsafe.Pointer(&gdt[0]))
	} else {
		C.build_32bit_gdt(unsafe.Pointer(&gdt[0]))
	}

	BootGdt := platform.DescriptorValue{
		Base:  uint64(gdt_addr),
		Limit: uint16(platform.PageSize),
	}
	err = vcpu.SetDescriptor(platform.GDT, BootGdt, true)
	if err != nil {
		return err
	}

	// Set a null IDT.
	BootIdt := platform.DescriptorValue{
		Base:  0,
		Limit: 0,
	}
	err = vcpu.SetDescriptor(platform.IDT, BootIdt, true)
	if err != nil {
		return err
	}

	// Enable protected-mode.
	// This does not set any flags (e.g. paging) beyond the
	// protected mode flag. This is according to Linux entry
	// protocol for 32-bit protected mode.
	cr0, err := vcpu.GetControlRegister(platform.CR0)
	if err != nil {
		return err
	}
	cr0 = cr0 | (1 << 0) // Protected mode.
	err = vcpu.SetControlRegister(platform.CR0, cr0, true)
	if err != nil {
		return err
	}

	// Always have the PAE bit set.
	cr4, err := vcpu.GetControlRegister(platform.CR4)
	if err != nil {
		return err
	}
	cr4 = cr4 | (1 << 5) // PAE enabled.
	err = vcpu.SetControlRegister(platform.CR4, cr4, true)
	if err != nil {
		return err
	}

	// For 64-bit kernels, we need to enable long mode,
	// and load an identity page table. This will require
	// only a page of pages, as we use huge page sizes.
	if is_64bit {
		// Create our page tables.
		pde_addr, pde, err := model.Allocate(
			machine.MemoryTypeUser,
			0,                 // Start.
			model.Max(),       // End.
			platform.PageSize, // Size.
			false)             // From bottom.
		if err != nil {
			return err
		}
		pgd_addr, pgd, err := model.Allocate(
			machine.MemoryTypeUser,
			0,                 // Start.
			model.Max(),       // End.
			platform.PageSize, // Size.
			false)             // From bottom.
		if err != nil {
			return err
		}
		pml4_addr, pml4, err := model.Allocate(
			machine.MemoryTypeUser,
			0,                 // Start.
			model.Max(),       // End.
			platform.PageSize, // Size.
			false)             // From bottom.
		if err != nil {
			return err
		}

		C.build_pde(unsafe.Pointer(&pde[0]), platform.PageSize)
		C.build_pgd(unsafe.Pointer(&pgd[0]), C.__u64(pde_addr), platform.PageSize)
		C.build_pml4(unsafe.Pointer(&pml4[0]), C.__u64(pgd_addr), platform.PageSize)

		log.Printf("loader: Created PDE @ %08x.", pde_addr)
		log.Printf("loader: Created PGD @ %08x.", pgd_addr)
		log.Printf("loader: Created PML4 @ %08x.", pml4_addr)

		// Set our newly build page table.
		err = vcpu.SetControlRegister(
			platform.CR3,
			platform.ControlRegisterValue(pml4_addr),
			true)
		if err != nil {
			return err
		}

		// Enable long mode.
		efer, err := vcpu.GetControlRegister(platform.EFER)
		if err != nil {
			return err
		}
		efer = efer | (1 << 8) // Long-mode enable.
		err = vcpu.SetControlRegister(platform.EFER, efer, true)
		if err != nil {
			return err
		}

		// Enable paging.
		cr0, err = vcpu.GetControlRegister(platform.CR0)
		if err != nil {
			return err
		}
		cr0 = cr0 | (1 << 31) // Paging enable.
		err = vcpu.SetControlRegister(platform.CR0, cr0, true)
		if err != nil {
			return err
		}
	}

	// NOTE: For 64-bit kernels, we need to enable
	// real 64-bit mode. This means that the L bit in
	// the segments must be one, the Db bit must be
	// zero, and we set the LME bit in EFER (above).
	var lVal uint8
	var dVal uint8
	if is_64bit {
		lVal = 1
		dVal = 0
	} else {
		lVal = 0
		dVal = 1
	}

	// Load the VMCS segments.
	//
	// NOTE: These values are loaded into the VMCS
	// registers and are expected to match the descriptors
	// we've used above. Unfortunately the API format doesn't
	// match, so we need to duplicate some work here. Ah, well
	// at least the below serves as an explanation for what
	// the magic numbers in GDT_ENTRY() above mean.
	BootCs := platform.SegmentValue{
		Base:     0,
		Limit:    0xffffffff,
		Selector: uint16(C.BootCsSelector), // @ 0x10
		Dpl:      0,                        // Privilege level (kernel).
		Db:       dVal,                     // 32-bit segment?
		G:        1,                        // Granularity (page).
		S:        1,                        // As per BOOT_CS (code/data).
		L:        lVal,                     // 64-bit extension.
		Type:     0xb,                      // As per BOOT_CS (access must be set).
		Present:  1,
	}
	BootDs := platform.SegmentValue{
		Base:     0,
		Limit:    0xffffffff,
		Selector: uint16(C.BootDsSelector), // @ 0x18
		Dpl:      0,                        // Privilege level (kernel).
		Db:       1,                        // 32-bit segment?
		G:        1,                        // Granularity (page).
		S:        1,                        // As per BOOT_DS (code/data).
		L:        0,                        // 64-bit extension.
		Type:     0x3,                      // As per BOOT_DS (access must be set).
		Present:  1,
	}
	BootTr := platform.SegmentValue{
		Base:     0,
		Limit:    0xffffffff,
		Selector: uint16(C.BootTrSelector), // @ 0x20
		Dpl:      0,                        // Privilege level (kernel).
		Db:       1,                        // 32-bit segment?
		G:        1,                        // Granularity (page).
		S:        0,                        // As per BOOT_TR (system).
		L:        0,                        // 64-bit extension.
		Type:     0xb,                      // As per BOOT_TR.
		Present:  1,
	}

	err = vcpu.SetSegment(platform.CS, BootCs, true)
	if err != nil {
		return err
	}

	err = vcpu.SetSegment(platform.DS, BootDs, true)
	if err != nil {
		return err
	}

	err = vcpu.SetSegment(platform.ES, BootDs, true)
	if err != nil {
		return err
	}

	err = vcpu.SetSegment(platform.FS, BootDs, true)
	if err != nil {
		return err
	}

	err = vcpu.SetSegment(platform.GS, BootDs, true)
	if err != nil {
		return err
	}

	err = vcpu.SetSegment(platform.SS, BootDs, true)
	if err != nil {
		return err
	}

	err = vcpu.SetSegment(platform.TR, BootTr, true)
	if err != nil {
		return err
	}

	// Create our boot parameters.
	boot_addr, boot_data, err := model.Allocate(
		machine.MemoryTypeUser,
		0,                 // Start.
		model.Max(),       // End.
		platform.PageSize, // Size.
		false)             // From bottom.
	if err != nil {
		return err
	}
	err = SetupLinuxBootParams(
		model,
		boot_data,
		orig_boot_data,
		cmdline_addr,
		initrd_addr,
		initrd_len)
	if err != nil {
		return err
	}

	// Set our registers.
	// This is according to the Linux 32-bit boot protocol.
	log.Printf("loader: boot_params @ %08x.", boot_addr)
	err = vcpu.SetRegister(platform.RSI, platform.RegisterValue(boot_addr))
	if err != nil {
		return err
	}

	err = vcpu.SetRegister(platform.RBP, 0)
	if err != nil {
		return err
	}

	err = vcpu.SetRegister(platform.RDI, 0)
	if err != nil {
		return err
	}

	err = vcpu.SetRegister(platform.RBX, 0)
	if err != nil {
		return err
	}

	// Jump to our entry point.
	err = vcpu.SetRegister(platform.RIP, platform.RegisterValue(entry_point))
	if err != nil {
		return err
	}

	// Make sure interrupts are disabled.
	// This will actually clear out all other flags.
	rflags, err := vcpu.GetRegister(platform.RFLAGS)
	if err != nil {
		return err
	}
	rflags = rflags &^ (1 << 9) // Interrupts off.
	rflags = rflags | (1 << 1)  // Reserved 1.
	err = vcpu.SetRegister(
		platform.RFLAGS,
		platform.RegisterValue(rflags))
	if err != nil {
		return err
	}

	// We're done.
	return nil
}
