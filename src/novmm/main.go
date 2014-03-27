package main

import (
    "flag"
    "log"
    "novmm/control"
    "novmm/loader"
    "novmm/machine"
    "novmm/platform"
    "os"
    "os/signal"
    "syscall"
)

// Our control server.
var control_fd = flag.Int("controlfd", -1, "bound control socket")

// Machine specifications.
var vcpu_data = flag.String("vcpus", "[]", "list of vcpu states")
var device_data = flag.String("devices", "[]", "list of device states")

// Functional flags.
var eventfds = flag.Bool("eventfds", false, "enable eventfds")
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

func main() {
    // Parse all command line options.
    flag.Parse()

    // Create VM.
    vm, err := platform.NewVm()
    if err != nil {
        log.Fatal(err)
    }
    defer vm.Dispose()
    if *eventfds {
        vm.EnableEventFds()
    }

    // Create the machine model.
    model, err := machine.NewModel(vm)
    if err != nil {
        log.Fatal(err)
    }

    // Load all vcpus.
    vcpus, err := vm.LoadVcpus([]byte(*vcpu_data))
    if err != nil {
        log.Fatal(err)
    }
    if len(vcpus) == 0 {
        log.Fatal(NoVcpus)
    }

    // Enable stepping if requested.
    if *step {
        for _, vcpu := range vcpus {
            vcpu.SetStepping(true)
        }
    }

    // Load all devices.
    proxy, err := model.LoadDevices(vm, []byte(*device_data), *debug)
    if err != nil {
        log.Fatal(err)
    }

    // Load given kernel and initrd.
    var sysmap loader.SystemMap
    var convention *loader.Convention

    if *vmlinux != "" {
        sysmap, convention, err = loader.LoadLinux(
            vcpus[0],
            model,
            *boot_params,
            *vmlinux,
            *initrd,
            *cmdline,
            *system_map)
        if err != nil {
            log.Fatal(err)
        }
    }

    // Create our tracer with the map and convention.
    tracer := loader.NewTracer(sysmap, convention)
    if *trace {
        tracer.Enable()
    }

    // Start all VCPUs.
    // None of these will actually come online
    // until the primary VCPU below delivers the
    // appropriate IPI to start them up.
    vcpu_err := make(chan error)
    for _, vcpu := range vcpus {
        go func(vcpu *platform.Vcpu) {
            defer vcpu.Dispose()
            err := Loop(vm, vcpu, model, tracer)
            vcpu_err <- err
        }(vcpu)
    }

    // Create our RPC server.
    control, err := control.NewControl(*control_fd, *real_init, model, vm, tracer, proxy)
    if err != nil {
        log.Fatal(err)
    }
    go control.Serve()

    // Wait until we get a TERM signal, or all the VCPUs are dead.
    vcpus_alive := len(vcpus)
    signals := make(chan os.Signal, 1)
    signal.Notify(signals, syscall.SIGTERM)

    for {
        select {
        case err := <-vcpu_err:
            vcpus_alive -= 1
            if err != nil {
                log.Printf("Vcpu died: %s", err.Error())
            }
        case sig := <-signals:
            switch sig {
            case syscall.SIGTERM:
                os.Exit(0)
            }
        }

        // Everything died?
        if vcpus_alive == 0 {
            log.Fatal(NoVcpus)
        }
    }
}
