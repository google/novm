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
	"fmt"
	"log"
	"novmm/platform"
	"strings"
)

type Tracer struct {
	sysmap     SystemMap
	convention *Convention
	last_addr  platform.Vaddr
	last_fname string
	enabled    bool
}

func NewTracer(sysmap SystemMap, convention *Convention) *Tracer {
	return &Tracer{
		sysmap:     sysmap,
		convention: convention,
		last_addr:  0,
		last_fname: "",
		enabled:    false,
	}
}

func (tracer *Tracer) Enable() {
	tracer.enabled = true
}

func (tracer *Tracer) Disable() {
	tracer.enabled = false
}

func (tracer *Tracer) IsEnabled() bool {
	return tracer.enabled
}

func (tracer *Tracer) toPaddr(
	vcpu *platform.Vcpu,
	reg platform.RegisterValue) string {

	phys_addr, valid, _, _, err := vcpu.Translate(platform.Vaddr(reg))
	if err != nil {
		return "%x->??"
	}

	if valid {
		return fmt.Sprintf("%x->%x", reg, phys_addr)
	}

	return fmt.Sprintf("%x", reg)
}

func (tracer *Tracer) Trace(vcpu *platform.Vcpu, step bool) error {

	// Are we on?
	if !tracer.enabled {
		return nil
	}

	// Get the current instruction.
	addr, err := vcpu.GetRegister(tracer.convention.instruction)
	if err != nil {
		return err
	}

	// Skip duplicates (only if stepping is on).
	if step && platform.Vaddr(addr) == tracer.last_addr {
		return nil
	}

	// Lookup the current instruction.
	var fname string
	var offset uint64
	if tracer.sysmap != nil {
		fname, offset = tracer.sysmap.Lookup(platform.Vaddr(addr))
	}

	// Get the stack depth.
	stack, err := vcpu.GetRegister(tracer.convention.stack)
	if err != nil {
		return err
	}

	// Print the return value if applicable.
	if step &&
		fname != tracer.last_fname &&
		tracer.last_addr != 0 {

		rval, err := vcpu.GetRegister(tracer.convention.rvalue)
		if err != nil {
			return err
		}
		log.Printf("  trace: [%08x] %s => %s ?",
			stack,
			tracer.last_fname,
			tracer.toPaddr(vcpu, rval))

		// Save the current.
		tracer.last_fname = fname
	}

	// Get a physical address string.
	rip_phys_str := tracer.toPaddr(vcpu, addr)

	if fname != "" {
		if offset == 0 {
			num_args := len(tracer.convention.arguments)
			arg_vals := make([]string, num_args, num_args)
			for i, reg := range tracer.convention.arguments {
				reg_val, err := vcpu.GetRegister(reg)
				if err != nil {
					arg_vals[i] = fmt.Sprintf("??")
					continue
				}
				arg_vals[i] = tracer.toPaddr(vcpu, reg_val)
			}
			log.Printf("  trace: [%08x] %s:%s(%s)",
				stack,
				fname,
				rip_phys_str,
				strings.Join(arg_vals, ","))
		} else {
			log.Printf("  trace: [%08x] %s:%s ... +%x",
				stack,
				fname,
				rip_phys_str,
				offset)
		}
	} else {
		log.Printf("  trace: ??:%s", rip_phys_str)
	}

	// We're okay.
	tracer.last_addr = platform.Vaddr(addr)
	return nil
}
