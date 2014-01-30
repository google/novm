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
    if process == nil {
        out.Written = -1
        return nil
    }

    // Is this a close?
    if write.Data == nil {
        process.cond.L.Lock()
        if process.stdin != nil {
            process.stdin.Close()
            process.stdin = nil
        }
        process.cond.L.Unlock()

        out.Written = 0
        return nil
    }

    // Push the write.
    var err error
    out.Written, err = process.stdin.Write(write.Data)
    if err != nil {
        out.Written = -1
    }

    return nil
}
