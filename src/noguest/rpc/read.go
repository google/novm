package rpc

type ReadCommand struct {

    // The relevant pid.
    Pid int `json:"pid"`

    // How much to read?
    N   uint `json:"n"`
}

type ReadResult struct {

    // The data read.
    Data []byte `json:"data"`
}

func (server *Server) Read(
    read *ReadCommand,
    result *ReadResult) error {

    process := server.lookup(read.Pid)
    if process == nil {
        result.Data = []byte{}
        return nil
    }

    // Read available data.
    buffer := make([]byte, read.N, read.N)
    n, err := process.terminal.Read(buffer)
    if n > 0 {
        result.Data = buffer[:n]
    } else {
        result.Data = []byte{}
    }

    return err
}
