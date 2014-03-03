package main

import (
    "net/rpc"
    "net/rpc/jsonrpc"
    "noguest/protocol"
)

func (control *Control) barrier() {
    buffer := make([]byte, 1, 1)

    // Send our control byte to noguest.
    // This essentially controls how the guest
    // will proceed during execution. If it is the
    // real init process, it will wait for run commands
    // and execute the given processes inside the VM.
    // If it is not the real init process, it will fork
    // and execute the real init before starting to
    // process any other RPC commands.
    if control.real_init {
        buffer[0] = protocol.NoGuestCommandRealInit
    } else {
        buffer[0] = protocol.NoGuestCommandFakeInit
    }
    n, err := control.proxy.Write(buffer)
    if n != 1 {
        // Can't send anything?
        control.client_err = InternalGuestError
        return
    }

    // REad our control byte back.
    n, err = control.proxy.Read(buffer)
    if n == 1 && err == nil {
        switch buffer[0] {
        case protocol.NoGuestStatusOkay:
            control.client_err = nil
            control.client_codec = jsonrpc.NewClientCodec(control.proxy)
            control.client = rpc.NewClientWithCodec(control.client_codec)
            break
        case protocol.NoGuestStatusFailed:
            // Something went horribly wrong.
            control.client_err = InternalGuestError
            return
        default:
            // This isn't good, who knows what happened?
            control.client_err = protocol.UnknownStatus
            return
        }
    } else if err != nil {
        // An actual error.
        control.client_err = err
    }
}

func (control *Control) ready() (*rpc.Client, error) {
    control.client_once.Do(control.barrier)
    return control.client, control.client_err
}
