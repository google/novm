package rpc

import (
    "bytes"
    "os"
    "sync"
    "syscall"
    "time"
    "unsafe"
)

/*
#define _XOPEN_SOURCE
#define _GNU_SOURCE
#include <stdlib.h>
#include <fcntl.h>
*/
import "C"

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

    // Open a master terminal Fd.
    fd, err := C.posix_openpt(syscall.O_RDWR | syscall.O_NOCTTY)
    if fd == C.int(-1) && err != nil {
        // Out of FDs?
        result.Pid = -1
        return err
    }

    // Save our master.
    terminal := os.NewFile(uintptr(fd), "ptmx")

    // Try to grant and unlock the PT.
    r, err := C.grantpt(C.int(terminal.Fd()))
    if r != C.int(0) && err != nil {
        terminal.Close()
        result.Pid = -1
        return err
    }
    r, err = C.unlockpt(C.int(terminal.Fd()))
    if r != C.int(0) && err != nil {
        terminal.Close()
        result.Pid = -1
        return err
    }

    // Get the terminal name.
    buf := make([]byte, 1024, 1024)
    r, err = C.ptsname_r(
        C.int(terminal.Fd()),
        (*C.char)(unsafe.Pointer(&buf[0])),
        1024)
    if r != C.int(0) && err != nil {
        terminal.Close()
        result.Pid = -1
        return err
    }

    // Open the slave terminal.
    n := bytes.Index(buf, []byte{0})
    slave_pts := string(buf[:n])
    slave, err := os.OpenFile(slave_pts, syscall.O_RDWR, 0)
    if err != nil {
        terminal.Close()
        result.Pid = -1
        return err
    }

    // Start the process.
    proc_attr := &os.ProcAttr{
        Dir:   command.Cwd,
        Env:   command.Environment,
        Files: []*os.File{slave, slave, slave},
        Sys: &syscall.SysProcAttr{
            Setsid:  true,
            Setctty: true,
            Ctty:    0,
        },
    }
    proc, err := os.StartProcess(
        command.Command[0],
        command.Command,
        proc_attr)

    // Close our slave.
    slave.Close()

    // Unable to start?
    if err != nil {
        terminal.Close()
        result.Pid = -1
        return err
    }

    // Create our process.
    process := &Process{
        terminal:  terminal,
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
