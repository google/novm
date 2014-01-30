package main

import (
    "bufio"
    "encoding/json"
    "net/rpc"
    "net/rpc/jsonrpc"
    noguest "noguest/rpc"
    "novmm/machine"
    "novmm/platform"
    "os"
    "strings"
    "syscall"
)

type DataEvent struct {

    // The data.
    Data []byte `json:"data"`

    // Should the file be closed?
    Close bool `json:"closed"`

    // Is this from stderr?
    Stderr bool `json:"stderr"`
}

func handleConn(
    conn_fd int,
    server *rpc.Server,
    client *rpc.Client) {

    control_file := os.NewFile(uintptr(conn_fd), "control")

    // Read single header.
    reader := bufio.NewReader(control_file)
    header, err := reader.ReadString('\n')
    if err != nil {
        control_file.Write([]byte(err.Error()))
        return
    }

    header = strings.TrimSpace(header)

    // We read a special header before diving into RPC
    // mode. This is because for the novmrun case, we turn
    // the socket into a stream of input/output events.
    // These are simply JSON serialized versions of the
    // events for the guest RPC interface.

    if header == "NOVM RUN" {

        decoder := json.NewDecoder(reader)
        encoder := json.NewEncoder(control_file)
        var start noguest.StartCommand
        err := decoder.Decode(&start)
        if err != nil {
            // Poorly encoded command.
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

        // Encode the result.
        encoder.Encode(&result)

        pid := result.Pid
        finished := make(chan error)

        // Wait for the process to exit.
        go func() {
            wait := noguest.WaitCommand{
                Pid: pid,
            }
            var wait_result noguest.WaitResult
            err := client.Call("Server.Wait", &wait, &wait_result)
            encoder.Encode(&wait_result)
            finished <- err
        }()

        // Read from stdout & stderr.
        read_fn := func(stderr bool) {
            read := noguest.ReadCommand{
                Pid:    pid,
                Stderr: stderr,
                N:      4096,
            }
            data_event := &DataEvent{
                Stderr: stderr,
                Close:  false,
            }
            var read_result noguest.ReadResult
            var err error
            for {
                err = client.Call("Server.Read", &read, &read_result)
                data_event.Data = read_result.Data
                if err != nil || read_result.Data == nil {
                    data_event.Close = true
                }
                encoder.Encode(&data_event)
                if data_event.Close {
                    break
                }
            }
            finished <- err
        }
        go read_fn(false)
        go read_fn(true)

        // Write to stdin.
        go func() {
            write := noguest.WriteCommand{
                Pid: pid,
            }
            buffer := make([]byte, 4096, 4096)
            var write_result noguest.WriteResult
            for {
                n, err := reader.Read(buffer)
                if n > 0 {
                    write.Data = buffer[:n]
                }
                for n > 0 {
                    err := client.Call("Server.Write", &write, &write_result)
                    if err != nil {
                        finished <- err
                        break
                    }
                    write.Data = write.Data[write_result.Written:]
                    n -= write_result.Written
                }
                if err != nil {
                    finished <- err
                    break
                }
            }
        }()

        // Wait for all the above to finish.
        <-finished
        <-finished
        <-finished
        <-finished

    } else if header == "NOVM RPC" {

        // Run as JSON RPC connection.
        codec := jsonrpc.NewServerCodec(control_file)
        server.ServeCodec(codec)
    }
}

func serveControl(
    control_fd int,
    vm *platform.Vm,
    proxy machine.Proxy) {

    // Bind our rpc server.
    server := rpc.NewServer()

    // Bind our client.
    codec := jsonrpc.NewClientCodec(proxy)
    client := rpc.NewClientWithCodec(codec)

    // Accept clients.
    for {
        nfd, _, err := syscall.Accept(control_fd)
        if err == nil {
            go handleConn(nfd, server, client)
        }
    }
}
