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

package loader

import (
	"io/ioutil"
	"log"
	"novmm/machine"
	"novmm/platform"
	"strconv"
	"strings"
)

type LinuxSystemMap struct {
	defined []platform.Vaddr
	symbols map[platform.Vaddr]string
	cache   map[platform.Vaddr]platform.Vaddr
}

func LoadLinuxSystemMap(
	system_map string) (SystemMap, error) {

	// No map provided.
	if system_map == "" {
		return nil, nil
	}

	// Read the file.
	map_data, err := ioutil.ReadFile(system_map)
	if err != nil {
		return nil, err
	}

	// Create our new map.
	sysmap := new(LinuxSystemMap)
	sysmap.defined = make([]platform.Vaddr, 0, 0)
	sysmap.symbols = make(map[platform.Vaddr]string)
	sysmap.cache = make(map[platform.Vaddr]platform.Vaddr)

	// Extract all symbols.
	log.Printf("loader: Reading symbols (%d bytes)...", len(map_data))

	add_symbol := func(line []byte) {
		// Format: <address> <type> <name>
		parts := strings.SplitN(string(line), " ", 3)
		if len(parts) != 3 {
			return
		}

		// Parse the address.
		addr, err := strconv.ParseUint(parts[0], 16, 64)
		if err != nil {
			return
		}

		// Save the symbol.
		sysmap.defined = append(sysmap.defined, platform.Vaddr(addr))
		sysmap.symbols[platform.Vaddr(addr)] = parts[2]
	}

	start_i := 0
	end_i := 0
	for end_i = 0; end_i < len(map_data); end_i += 1 {
		if map_data[end_i] == '\n' {
			add_symbol(map_data[start_i:end_i])
			start_i = (end_i + 1)
		}
	}
	if start_i != end_i && start_i < end_i {
		add_symbol(map_data[start_i:end_i])
	}

	// Return our map.
	log.Printf("loader: System map has %d entries.", len(sysmap.defined))
	return sysmap, nil
}

func (sysmap *LinuxSystemMap) Lookup(
	addr platform.Vaddr) (string, uint64) {

	// Bounds check.
	if sysmap == nil ||
		len(sysmap.defined) == 0 {
		return "", 0
	}

	// Check the cache.
	symaddr, ok := sysmap.cache[addr]
	if ok {
		return sysmap.symbols[symaddr], uint64(addr - symaddr)
	}

	// Do a binary search.
	min_index := 0
	max_index := len(sysmap.defined)
	for min_index < max_index {
		index := min_index + (max_index-min_index+1)/2
		if sysmap.defined[index] < addr {
			min_index = index
		} else if sysmap.defined[index] > addr {
			max_index = index - 1
		} else {
			min_index = index
			max_index = index
		}
	}

	// Check for invalid result.
	if sysmap.defined[min_index] > addr {
		return "", 0
	}

	// Cache the result.
	symaddr = sysmap.defined[min_index]
	sysmap.cache[addr] = symaddr

	// Return the result.
	return sysmap.symbols[symaddr], uint64(addr - symaddr)
}

func LoadLinux(
	vcpu *platform.Vcpu,
	model *machine.Model,
	boot_params string,
	vmlinux string,
	initrd string,
	cmdline string,
	system_map string) (SystemMap, *Convention, error) {

	// Read the boot_params.
	log.Print("loader: Reading kernel image...")
	kernel_data, err := ioutil.ReadFile(boot_params)
	log.Printf("loader: Kernel is %d bytes.", len(kernel_data))
	if err != nil {
		return nil, nil, err
	}
	// They may have passed the entire vmlinuz image as the
	// parameter here. That's okay, we do an efficient mmap
	// above. But we need to truncate the visible slice.
	boot_params_data := kernel_data[0:platform.PageSize]

	// Load the kernel.
	log.Print("loader: Reading kernel binary...")
	vmlinux_data, err := ioutil.ReadFile(vmlinux)
	log.Printf("loader: Kernel binary is %d bytes.", len(vmlinux_data))
	if err != nil {
		return nil, nil, err
	}

	// Load the ramdisk.
	log.Print("loader: Reading ramdisk...")
	initrd_data, err := ioutil.ReadFile(initrd)
	log.Printf("loader: Ramdisk is %d bytes.", len(initrd_data))
	if err != nil {
		return nil, nil, err
	}

	// Load the system map.
	log.Print("loader: Loading system map...")
	sysmap, err := LoadLinuxSystemMap(system_map)
	if err != nil {
		return nil, nil, err
	}

	// Load the kernel into memory.
	log.Print("loader: Loading kernel...")
	entry_point, is_64bit, err := ElfLoad(vmlinux_data, model)
	if err != nil {
		return nil, nil, err
	}
	if is_64bit {
		log.Print("loader: 64-bit kernel found.")
	} else {
		log.Print("loader: 32-bit kernel found.")
	}
	log.Printf("loader: Entry point is %08x.", entry_point)

	// Set our calling convention.
	var convention *Convention
	if is_64bit {
		convention = &Linux64Convention
	} else {
		convention = &Linux32Convention
	}

	// Load the cmdline.
	// NOTE: Here we create a full page with
	// trailing zeros. This is the expected form
	// for the command line.
	full_cmdline := make(
		[]byte,
		platform.PageSize,
		platform.PageSize)
	copy(full_cmdline, []byte(cmdline))

	cmdline_addr, err := model.MemoryMap.Load(
		platform.Paddr(0),
		model.Max(),
		full_cmdline,
		false)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("loader: cmdline @ %08x: %s",
		cmdline_addr,
		cmdline)

	// Load the initrd.
	initrd_addr, err := model.MemoryMap.Load(
		platform.Paddr(0),
		model.Max(),
		initrd_data,
		true)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("loader: initrd @ %08x.", initrd_addr)

	// Create our setup page,
	// and initialize the VCPU.
	err = SetupLinux(
		vcpu,
		model,
		boot_params_data,
		entry_point,
		is_64bit,
		initrd_addr,
		uint64(len(initrd_data)),
		cmdline_addr)

	// Everything is okay.
	return sysmap, convention, err
}
