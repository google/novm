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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"novmm/control"
	"novmm/loader"
	"novmm/machine"
	"novmm/platform"
	"novmm/utils"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Our control server.
var control_fd = flag.Int("controlfd", -1, "bound control socket")

// Machine state.
var statefd = flag.Int("statefd", 0, "machine state file")

// Guest-related flags.
var real_init = flag.Bool("init", false, "real in-guest init?")

// Linux parameters.
var boot_params = flag.String("setup", "", "linux boot params (vmlinuz)")
var vmlinux = flag.String("vmlinux", "", "linux kernel binary (ELF)")
var initrd = flag.String("initrd", "", "initial ramdisk image")
var cmdline = flag.String("cmdline", "", "linux command line")
var system_map = flag.String("sysmap", "", "kernel symbol map")

// Debug parameters.
var step = flag.Bool("step", false, "step instructions")
var trace = flag.Bool("trace", false, "trace kernel symbols on exit")
var debug = flag.Bool("debug", false, "devices start debugging")
var paused = flag.Bool("paused", false, "start with model and vcpus paused")
var stop = flag.Bool("stop", false, "wait for a SIGCONT before running")

func restart(
	model *machine.Model,
	vm *platform.Vm,
	is_tracing bool,
	stop bool) error {

	// Get our binary.
	bin, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return err
	}
	_, err = os.Stat(bin)
	if err != nil {
		// If this is no longer the same binary, then the
		// kernel proc node will have "fixed" the symlink
		// to point to "/path (deleted)". This is mildly
		// annoying, as one would assume there would be a
		// better way of transmitting that information.
		if os.IsNotExist(err) && strings.HasSuffix(bin, " (deleted)") {
			bin = strings.TrimSuffix(bin, " (deleted)")
			_, err = os.Stat(bin)
		}
		if err != nil {
			return err
		}
	}

	// Create our state.
	state, err := control.SaveState(vm, model)
	if err != nil {
		return err
	}

	// Encode our state in a temporary file.
	// This is passed in to the new VMM as the statefd.
	// We unlink it immediately because we don't need to
	// access it by name, and can ensure it is cleaned up.
	// Note that the TempFile is normally opened CLOEXEC.
	// This means that need we need to perform a DUP in
	// order to get an FD that can pass to the child.
	state_file, err := ioutil.TempFile(os.TempDir(), "state")
	if err != nil {
		return err
	}
	defer state_file.Close()
	err = os.Remove(state_file.Name())
	if err != nil {
		return err
	}
	encoder := utils.NewEncoder(state_file)
	err = encoder.Encode(&state)
	if err != nil {
		return err
	}
	_, err = state_file.Seek(0, 0)
	if err != nil {
		return err
	}
	state_fd, err := syscall.Dup(int(state_file.Fd()))
	if err != nil {
		return err
	}
	defer syscall.Close(state_fd)

	// Prepare to reexec.
	cmd := []string{
		os.Args[0],
		fmt.Sprintf("-controlfd=%d", *control_fd),
		fmt.Sprintf("-statefd=%d", state_fd),
		fmt.Sprintf("-trace=%t", is_tracing),
		fmt.Sprintf("-paused=%t", *paused),
		fmt.Sprintf("-stop=%t", stop),
	}

	return syscall.Exec(bin, cmd, os.Environ())
}

func main() {
	// Start processing signals.
	// Our setup can take a little while, so we
	// want to ensure we aren't using the default
	// handlers from the beginning.
	signals := make(chan os.Signal, 1)
	signal.Notify(
		signals,
		utils.SigShutdown,
		utils.SigRestart,
		utils.SigSpecialRestart)

	// Parse all command line options.
	flag.Parse()

	// Are we doing a special restart?
	// This will STOP the current process, and
	// wait for a CONT signal before resuming.
	// The STOP signal is not maskable, so the
	// runtime isn't capable of preventing this.
	// The whole point of this restart is as follows:
	//   * killall -USR2 novmm
	//   * upgrade kvm
	//   * killall -CONT novmm
	if *stop {
		syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	}

	// Create VM.
	vm, err := platform.NewVm()
	if err != nil {
		utils.Die(err)
	}
	defer vm.Dispose()

	// Create the machine model.
	model, err := machine.NewModel(vm)
	if err != nil {
		utils.Die(err)
	}

	// Load our machine state.
	state_file := os.NewFile(uintptr(*statefd), "state")
	decoder := utils.NewDecoder(state_file)
	state := new(control.State)
	err = decoder.Decode(&state)
	if err != nil {
		utils.Die(err)
	}

	// We're done with the state file.
	state_file.Close()

	// Load all devices.
	log.Printf("Creating devices...")
	proxy, err := model.CreateDevices(vm, state.Devices, *debug)
	if err != nil {
		utils.Die(err)
	}

	// Load all vcpus.
	log.Printf("Creating vcpus...")
	vcpus, err := vm.CreateVcpus(state.Vcpus)
	if err != nil {
		utils.Die(err)
	}
	if len(vcpus) == 0 {
		utils.Die(NoVcpus)
	}

	// Load all model state.
	log.Printf("Loading model...")
	err = model.Load(vm)
	if err != nil {
		utils.Die(err)
	}

	// Pause all devices and vcpus if requested.
	if *paused {
		err = model.Pause(true)
		if err != nil {
			utils.Die(err)
		}
		err = vm.Pause(true)
		if err != nil {
			utils.Die(err)
		}
	}

	// Enable stepping if requested.
	if *step {
		for _, vcpu := range vcpus {
			vcpu.SetStepping(true)
		}
	}

	// Remember whether or not this is a load.
	// If it's a load, then we have to sync the
	// control interface. If it's not, then we
	// should skip the control interface sync.
	is_load := false

	// Load given kernel and initrd.
	var sysmap loader.SystemMap
	var convention *loader.Convention

	if *vmlinux != "" {
		log.Printf("Loading linux...")
		sysmap, convention, err = loader.LoadLinux(
			vcpus[0],
			model,
			*boot_params,
			*vmlinux,
			*initrd,
			*cmdline,
			*system_map)
		if err != nil {
			utils.Die(err)
		}

		// This is a fresh boot.
		is_load = true
	}

	// Create our tracer with the map and convention.
	tracer := loader.NewTracer(sysmap, convention)
	if *trace {
		tracer.Enable()
	}

	// Create our RPC server.
	log.Printf("Starting control server...")
	control, err := control.NewControl(
		*control_fd,
		*real_init,
		model,
		vm,
		tracer,
		proxy,
		is_load)
	if err != nil {
		utils.Die(err)
	}
	go control.Serve()

	// Start all VCPUs.
	// None of these will actually come online
	// until the primary VCPU below delivers the
	// appropriate IPI to start them up.
	log.Printf("Starting vcpus...")
	vcpu_err := make(chan error)
	for _, vcpu := range vcpus {
		go func(vcpu *platform.Vcpu) {
			err := Loop(vm, vcpu, model, tracer)
			vcpu_err <- err
		}(vcpu)
	}

	// Wait until we get a TERM signal, or all the VCPUs are dead.
	// If we receive a HUP signal, then we will re-exec with the
	// appropriate device state and vcpu state. This is essentially
	// a live upgrade (i.e. the binary has been replaced, we rerun).
	vcpus_alive := len(vcpus)

	for {
		select {
		case err := <-vcpu_err:
			vcpus_alive -= 1
			if err != nil {
				log.Printf("Vcpu died: %s", err.Error())
			}
		case sig := <-signals:
			switch sig {
			case utils.SigShutdown:
				log.Printf("Shutdown.")
				os.Exit(0)

			case utils.SigRestart:
				fallthrough
			case utils.SigSpecialRestart:

				// Make sure we have control sync'ed.
				_, err := control.Ready()
				if err != nil {
					utils.Die(err)
				}

				// This is a bit of a special case.
				// We don't log a fatal message here,
				// but rather unpause and keep going.
				err = restart(
					model,
					vm,
					tracer.IsEnabled(),
					sig == utils.SigSpecialRestart)
				log.Printf("Restart failed: %s", err.Error())
			}
		}

		// Everything died?
		if vcpus_alive == 0 {
			utils.Die(NoVcpus)
		}
	}
}
