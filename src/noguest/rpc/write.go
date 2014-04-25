package rpc

type WriteCommand struct {

    // The relevant pid.
    Pid int `json:"pid"`

    // The write.
    Data []byte `json:"data"`
}

type WriteResult struct {

    // How much was written?
    Written int `json:"n"`
}

func (server *Server) Write(
    write *WriteCommand,
    out *WriteResult) error {

    process := server.lookup(write.Pid)
    if process == nil || write.Data == nil {
        out.Written = -1
        return nil
    }

    // Push the write.
    for len(write.Data) > 0 {
        n, err := process.input.Write(write.Data)
        out.Written += n
        write.Data = write.Data[n:]
        if err != nil {
            return err
        }
    }

    return nil
}
