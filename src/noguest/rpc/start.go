package rpc

import (
    "os"
    "sync"
    "time"
)

type StartCommand struct {

    // The command to run.
    Command []string `json:"command"`

    // The working directory.
    Cwd string `json:"cwd"`

    // The environment.
    Environment []string `json:"environment"`
}

type StartResult struct {

    // The resulting pid.
    Pid int `json:"pid"`
}

func (server *Server) Start(
    command *StartCommand,
    result *StartResult) error {

    stdin_r, stdin_w, err := os.Pipe()
    if err != nil {
        // Out of FDs?
        result.Pid = -1
        return err
    }
    stdout_r, stdout_w, err := os.Pipe()
    if err != nil {
        // Out of FDs?
        stdin_r.Close()
        stdin_w.Close()
        result.Pid = -1
        return err
    }
    stderr_r, stderr_w, err := os.Pipe()
    if err != nil {
        // Out of FDs?
        stdin_r.Close()
        stdin_w.Close()
        stdout_r.Close()
        stdout_w.Close()
        result.Pid = -1
        return err
    }

    proc_attr := &os.ProcAttr{
        Dir:   command.Cwd,
        Env:   command.Environment,
        Files: []*os.File{stdin_r, stdout_w, stderr_w},
    }

    proc, err := os.StartProcess(
        command.Command[0],
        command.Command,
        proc_attr)

    // Unable to start?
    if err != nil {
        result.Pid = -1
        return err
    }

    // Create our process.
    process := &Process{
        stdin:     stdin_w,
        stdout:    stdout_r,
        stderr:    stderr_r,
        starttime: time.Now(),
        cond:      sync.NewCond(&sync.Mutex{}),
    }

    // Save the pid.
    result.Pid = proc.Pid

    server.mutex.Lock()
    old_process := server.active[result.Pid]
    server.active[result.Pid] = process
    server.mutex.Unlock()

    if old_process != nil {
        old_process.close()
    }

    go server.wait()
    return nil
}
