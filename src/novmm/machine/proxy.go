package machine

import (
    "io"
)

//
// Proxy --
//
// The proxy is something that allows us to connect
// with the agent inside the VM. At the moment, this
// is only the virtio_console device. Theoretically,
// any of the devices could implement this interface
// if the agent supported it....
//

type Proxy interface {
    io.ReadWriteCloser
}
