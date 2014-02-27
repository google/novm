package main

import (
    "encoding/json"
    "net/rpc"
    "net/rpc/jsonrpc"
    noguest "noguest/rpc"
    "novmm/loader"
    "novmm/machine"
    "novmm/platform"
    "os"
    "sync"
    "syscall"
)

type Control struct {

    // The bound control fd.
    control_fd int

    // Our device model.
    model *machine.Model

    // Our underlying Vm object.
    vm  *platform.Vm

    // Our tracer.
    tracer *loader.Tracer

    // Our proxy to the in-guest agent.
    proxy machine.Proxy

    // Our bound client (to the in-guest agent).
    // NOTE: We have this setup as a lazy
    // function because the guest may take
    // some small amount of time before its
    // actually ready to process RPC requests.
    // But we don't want this to interfere
    // with our ability to process our host
    // side RPC requests.
    client_err   error
    client_once  sync.Once
    client_codec rpc.ClientCodec
    client       *rpc.Client
}

type VmSettings struct {
}

type TraceSettings struct {
    // Tracing?
    Enable bool `json:"enable"`
}

type VcpuSettings struct {
    // Which vcpu?
    Id  int `json:"id"`

    // Single stepping?
    Step bool `json:"step"`
}

func (control *Control) Vm(settings *VmSettings, ok *bool) error {
    *ok = true
    return nil
}

func (control *Control) Trace(settings *TraceSettings, ok *bool) error {
    if settings.Enable {
        control.tracer.Enable()
    } else {
        control.tracer.Disable()
    }
    *ok = true
    return nil
}

func (control *Control) Vcpu(settings *VcpuSettings, ok *bool) error {
    // A valid vcpu?
    vcpus := control.vm.GetVcpus()
    if settings.Id >= len(vcpus) {
        *ok = false
        return syscall.EINVAL
    }
    vcpu := vcpus[settings.Id]
    err := vcpu.SetStepping(settings.Step)
    *ok = (err == nil)
    return err
}

func (control *Control) handle(
    conn_fd int,
    server *rpc.Server) {

    control_file := os.NewFile(uintptr(conn_fd), "control")
    defer control_file.Close()

    // Read single header.
    // Our header is exactly 9 characters, and we
    // expect the last character to be a newline.
    // This is a simple plaintext protocol.
    header_buf := make([]byte, 9, 9)
    n, err := control_file.Read(header_buf)
    if n != 9 || header_buf[8] != '\n' {
        if err != nil {
            control_file.Write([]byte(err.Error()))
        } else {
            control_file.Write([]byte("invalid header"))
        }
        return
    }
    header := string(header_buf)

    // We read a special header before diving into RPC
    // mode. This is because for the novmrun case, we turn
    // the socket into a stream of input/output events.
    // These are simply JSON serialized versions of the
    // events for the guest RPC interface.

    if header == "NOVM RUN\n" {

        decoder := json.NewDecoder(control_file)
        encoder := json.NewEncoder(control_file)

        var start noguest.StartCommand
        err := decoder.Decode(&start)
        if err != nil {
            // Poorly encoded command.
            encoder.Encode(err.Error())
            return
        }

        // Grab our client.
        client, err := control.ready()
        if err != nil {
            encoder.Encode(err.Error())
            return
        }

        // Call start.
        result := noguest.StartResult{}
        err = client.Call("Server.Start", &start, &result)
        if err != nil {
            encoder.Encode(err.Error())
            return
        }

        // Save our pid.
        pid := result.Pid
        inputs := make(chan error)
        outputs := make(chan error)
        exitcode := make(chan int)

        // This indicates we're okay.
        encoder.Encode(nil)

        // Wait for the process to exit.
        go func() {
            wait := noguest.WaitCommand{
                Pid: pid,
            }
            var wait_result noguest.WaitResult
            err := client.Call("Server.Wait", &wait, &wait_result)
            if err != nil {
                exitcode <- 1
            } else {
                exitcode <- wait_result.Exitcode
            }
        }()

        // Read from stdout & stderr.
        go func() {
            read := noguest.ReadCommand{
                Pid: pid,
                N:   4096,
            }
            var read_result noguest.ReadResult
            for {
                err := client.Call("Server.Read", &read, &read_result)
                if err != nil {
                    inputs <- err
                    return
                }
                err = encoder.Encode(read_result.Data)
                if err != nil {
                    inputs <- err
                    return
                }
            }
        }()

        // Write to stdin.
        go func() {
            write := noguest.WriteCommand{
                Pid: pid,
            }
            var write_result noguest.WriteResult
            for {
                err := decoder.Decode(&write.Data)
                if err != nil {
                    outputs <- err
                    return
                }
                err = client.Call("Server.Write", &write, &write_result)
                if err != nil {
                    outputs <- err
                    return
                }
            }
        }()

        // Wait till exit.
        status := <-exitcode
        encoder.Encode(status)

        // Wait till EOF.
        <-inputs

        // Send a notice and close the socket.
        encoder.Encode(nil)

    } else if header == "NOVM RPC\n" {

        // Run as JSON RPC connection.
        codec := jsonrpc.NewServerCodec(control_file)
        server.ServeCodec(codec)
    }
}

func (control *Control) barrier() {
    buffer := make([]byte, 1, 1)
    n, err := control.proxy.Read(buffer)
    if n == 1 && err == nil {
        control.client_err = nil
        control.client_codec = jsonrpc.NewClientCodec(control.proxy)
        control.client = rpc.NewClientWithCodec(control.client_codec)
    } else if err != nil {
        control.client_err = err
    }
}

func (control *Control) ready() (*rpc.Client, error) {
    control.client_once.Do(control.barrier)
    return control.client, control.client_err
}

func (control *Control) serve() {

    // Bind our rpc server.
    server := rpc.NewServer()
    server.Register(control)

    for {
        // Accept clients.
        nfd, _, err := syscall.Accept(control.control_fd)
        if err == nil {
            go control.handle(nfd, server)
        }
    }
}

func NewControl(
    control_fd int,
    model *machine.Model,
    vm *platform.Vm,
    tracer *loader.Tracer,
    proxy machine.Proxy) *Control {

    // Create our control object.
    control := new(Control)
    control.control_fd = control_fd
    control.vm = vm
    control.tracer = tracer
    control.proxy = proxy

    return control
}
