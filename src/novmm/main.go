package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "novmm/control"
    "novmm/loader"
    "novmm/machine"
    "novmm/platform"
    "os"
    "os/signal"
    "strings"
    "syscall"
)

// Our control server.
var control_fd = flag.Int("controlfd", -1, "bound control socket")

// Machine state.
var statefd = flag.Int("statefd", 0, "machine state file")

// Functional flags.
var eventfds = flag.Bool("eventfds", false, "enable eventfds")

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
var freakout = flag.Bool("panic", false, "panic on fatal error")

func restart(
    model *machine.Model,
    vm *platform.Vm,
    is_tracing bool) error {

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

    // Pause all vcpus.
    // (If we have to exit this function, we
    // ensure that the Vcpus will be unpaused).
    vcpus := vm.Vcpus()
    for _, vcpu := range vcpus {
        err := vcpu.Pause(false)
        if err != nil {
            return err
        }
        defer vcpu.Unpause(false)
    }

    // Collect our device and vcpu data.
    // Note that devices and vcpus contain all stepping
    // and debugging information, so it's not necessary
    // to replay those arguments (and would be wrong).
    vcpuinfo, err := vm.VcpuInfo()
    if err != nil {
        return err
    }
    deviceinfo, err := model.DeviceInfo(vm)
    if err != nil {
        return err
    }
    state := &control.State{
        Devices: deviceinfo,
        Vcpus:   vcpuinfo,
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
    json_encoder := json.NewEncoder(state_file)
    err = json_encoder.Encode(&state)
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
        fmt.Sprintf("-eventfds=%t", *eventfds),
        fmt.Sprintf("-trace=%t", is_tracing),
    }

    return syscall.Exec(bin, cmd, os.Environ())
}

func die(err error) {
    log.Fatal(err)
}

func main() {
    // Parse all command line options.
    flag.Parse()

    // Create VM.
    vm, err := platform.NewVm()
    if err != nil {
        die(err)
    }
    defer vm.Dispose()
    if *eventfds {
        vm.EnableEventFds()
    }

    // Create the machine model.
    model, err := machine.NewModel(vm)
    if err != nil {
        die(err)
    }

    // Load our machine state.
    state_file := os.NewFile(uintptr(*statefd), "state")
    json_decoder := json.NewDecoder(state_file)
    json_decoder.UseNumber()
    state := new(control.State)
    err = json_decoder.Decode(&state)
    if err != nil {
        die(err)
    }

    // We're done with the state file.
    state_file.Close()

    // Load all devices.
    proxy, err := model.LoadDevices(vm, state.Devices, *debug)
    if err != nil {
        die(err)
    }

    // Load all vcpus.
    vcpus, err := vm.LoadVcpus(state.Vcpus)
    if err != nil {
        die(err)
    }
    if len(vcpus) == 0 {
        die(NoVcpus)
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
        sysmap, convention, err = loader.LoadLinux(
            vcpus[0],
            model,
            *boot_params,
            *vmlinux,
            *initrd,
            *cmdline,
            *system_map)
        if err != nil {
            die(err)
        }

        // This is a fresh boot.
        is_load = true
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
    control, err := control.NewControl(
        *control_fd,
        *real_init,
        model,
        vm,
        tracer,
        proxy,
        is_load)

    if err != nil {
        die(err)
    }
    go control.Serve()

    // Wait until we get a TERM signal, or all the VCPUs are dead.
    // If we receive a HUP signal, then we will re-exec with the
    // appropriate device state and vcpu state. This is essentially
    // a live upgrade (i.e. the binary has been replaced, we rerun).
    vcpus_alive := len(vcpus)
    signals := make(chan os.Signal, 1)
    signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP)

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
                log.Printf("Shutdown.")
                os.Exit(0)

            case syscall.SIGHUP:
                // Make sure we have control sync'ed.
                _, err := control.Ready()
                if err != nil {
                    die(err)
                }

                // This is a bit of a special case.
                // We don't log a fatal message here,
                // but rather unpause and keep going.
                err = restart(model, vm, tracer.IsEnabled())
                log.Printf("Restart failed: %s", err.Error())
            }
        }

        // Everything died?
        if vcpus_alive == 0 {
            die(NoVcpus)
        }
    }
}
