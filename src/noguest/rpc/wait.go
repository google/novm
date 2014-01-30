package rpc

type WaitCommand struct {

    // The relevant pid.
    Pid int `json:"pid"`
}

type WaitResult struct {

    // The exit code.
    // (If > 0 then this event is an exit event).
    Exitcode int `json:"exitcode"`
}

func (server *Server) Wait(
    wait *WaitCommand,
    result *WaitResult) error {

    process := server.lookup(wait.Pid)
    if process == nil {
        result.Exitcode = -1
        return nil
    }

    process.wait()
    result.Exitcode = process.exitcode
    return nil
}
