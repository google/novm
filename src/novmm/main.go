package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "log"
    "novmm/loader"
    "novmm/machine"
    "novmm/platform"
)

// Machine specifications.
var vcpus = flag.Int("vcpus", 1, "number of guest VCPUs")
var memory = flag.Int("memory", 512, "guest memory size (in MB)")
var devices = flag.String("devices", "{}", "list of specified device")

// Linux parameters.
var boot_params = flag.String("setup", "", "linux boot params (vmlinuz)")
var vmlinux = flag.String("vmlinux", "", "linux kernel binary (ELF)")
var initrd = flag.String("initrd", "", "initial ramdisk image")
var cmdline = flag.String("cmdline", "", "linux command line")
var system_map = flag.String("sysmap", "", "kernel symbol map")

// Debug parameters.
var step = flag.Bool("step", false, "step instructions")
var trace = flag.Bool("trace", false, "trace kernel symbols on exit")

//
// All devices are passed in as a device list.
// This may or may not include device state so we
// could implement suspend/resume at a later point.
//
type DeviceList []machine.DeviceInfo

func decodeDeviceList(data string) DeviceList {

    // Decode a new object.
    json_decoder := json.NewDecoder(bytes.NewBuffer([]byte(data)))
    spec := make(DeviceList, 0, 0)
    err := json_decoder.Decode(&spec)
    if err != nil {
        log.Fatal(err)
    }

    // Return the result.
    return spec
}

func main() {
    // Parse all command line options.
    flag.Parse()

    // Sanity check flags.
    if *vcpus < 1 {
        log.Fatal(platform.InvalidVcpus)
    }
    if *memory < 1 {
        log.Fatal(platform.InvalidMemory)
    }
    if *vmlinux == "" {
        log.Fatal(NoKernelProvided)
    }

    // Decode our device specs.
    devices := decodeDeviceList(*devices)

    // Create VM.
    vm, err := platform.NewVm()
    if err != nil {
        log.Fatal(err)
    }
    defer vm.Dispose()

    // Create the machine model.
    model, err := machine.NewModel(
        vm,
        uint(*vcpus),
        uint64(*memory)*1024*1024)
    if err != nil {
        vm.Dump()
        log.Fatal(err)
    }

    // Create first VCPU.
    vcpu, err := vm.NewVcpu()
    if err != nil {
        vm.Dump()
        log.Fatal(err)
    }
    defer vcpu.Dispose()

    // Add all of our devices.
    for _, device := range devices {
        err := model.LoadDevice(&device)
        if err != nil {
            log.Fatal(err)
        }
    }

    // Load given kernel and initrd.
    sysmap, convention, err := loader.LoadLinux(
        vcpu,
        model,
        *boot_params,
        *vmlinux,
        *initrd,
        *cmdline,
        *system_map)
    if err != nil {
        vm.Dump()
        log.Fatal(err)
    }

    // Create our tracer with the map and convention.
    tracer := loader.NewTracer(sysmap, convention)
    if *trace {
        tracer.Enable()
    }

    // Start our model.
    err = model.Start()
    if err != nil {
        log.Fatal(err)
    }
    defer model.Stop()

    // Create and start additional VCPUs.
    // None of these will actually come online
    // until the primary VCPU below delivers the
    // appropriate IPI to start them up.
    for cpu := 1; cpu < *vcpus; cpu += 1 {
        other_vcpu, err := vm.NewVcpu()
        if err != nil {
            vm.Dump()
            log.Fatal(err)
        }

        // NOTE: It's completely possible that these
        // VCPUs will fail. Unlike the first CPU, we
        // don't fail the entire machine in that case.
        // (Although it's likely the OS is f*cked).
        go func() {
            Loop(other_vcpu, model, *step, tracer)
            other_vcpu.Dispose()
        }()
    }

    // Run.
    err = Loop(vcpu, model, *step, tracer)
    if err != nil {
        vm.Dump()
        log.Fatal(err)
    }
}
