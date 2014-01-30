package rpc

import (
    "os"
)

type ReadCommand struct {

    // The relevant pid.
    Pid int `json:"pid"`

    // Stdin or stderr?
    Stderr bool `json:"stderr"`

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
        result.Data = nil
        return nil
    }

    var file *os.File
    if read.Stderr {
        file = process.stderr
    } else {
        file = process.stdout
    }

    if file == nil {
        result.Data = nil
    } else {
        buffer := make([]byte, read.N, read.N)
        n, err := file.Read(buffer)
        if n > 0 {
            result.Data = buffer[:n]
        } else {
            process.cond.L.Lock()
            if read.Stderr {
                process.stderr.Close()
                process.stderr = nil
            } else {
                process.stdout.Close()
                process.stdout = nil
            }
            process.cond.L.Unlock()
            result.Data = nil
            return err
        }
    }

    return nil
}
