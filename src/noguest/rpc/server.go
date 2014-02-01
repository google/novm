package rpc

import (
    "net/rpc"
    "net/rpc/jsonrpc"
    "os"
    "sync"
    "syscall"
    "time"
)

type Process struct {

    // The pipes.
    stdin  *os.File
    stdout *os.File
    stderr *os.File

    // The start time.
    starttime time.Time

    // The exit time.
    exittime time.Time

    // Has this exited?
    exited bool

    // Our exitcode.
    exitcode int

    cond *sync.Cond
}

func (process *Process) wait() {
    process.cond.L.Lock()
    defer process.cond.L.Unlock()

    // Until the process is done.
    for !process.exited {
        process.cond.Wait()
    }
}

func (process *Process) setExitcode(exitcode int) {
    process.cond.L.Lock()
    defer process.cond.L.Unlock()

    // Set the exitcode.
    process.exited = true
    process.exitcode = exitcode
    process.exittime = time.Now()
    process.cond.Broadcast()
}

func (process *Process) close() {
    process.cond.L.Lock()
    defer process.cond.L.Unlock()

    if process.stdin != nil {
        process.stdin.Close()
        process.stdin = nil
    }
    if process.stdout != nil {
        process.stdout.Close()
        process.stdout = nil
    }
    if process.stderr != nil {
        process.stderr.Close()
        process.stderr = nil
    }

    // Simulate an exit.
    process.setExitcode(1)
}

type Server struct {

    // Active processes.
    active map[int]*Process

    // Is wait running?
    waiting bool

    // Our lock protects
    // access to the above map.
    mutex sync.Mutex
}

func (server *Server) clearStale() {
    server.mutex.Lock()
    defer server.mutex.Unlock()

    for pid, process := range server.active {
        // Has this exited more than a minute ago?
        if process.exited &&
            process.exittime.Sub(process.starttime) > time.Minute {
            delete(server.active, pid)
            process.close()
        }
    }
}

func (server *Server) clearPeriodic() {
    server.clearStale()
    time.AfterFunc(time.Minute, server.clearPeriodic)
}

func (server *Server) lookup(pid int) *Process {
    server.mutex.Lock()
    defer server.mutex.Unlock()
    return server.active[pid]
}

func (server *Server) wait() {

    server.mutex.Lock()
    if server.waiting {
        server.mutex.Unlock()
        return
    }
    server.waiting = true
    server.mutex.Unlock()

    var wstatus syscall.WaitStatus
    var rusage syscall.Rusage
    var last_run bool
    for {
        pid, err := syscall.Wait4(-1, &wstatus, 0, &rusage)
        if err != nil {
            if err == syscall.ECHILD {
                // Run once more to catch any races.
                server.mutex.Lock()
                if server.waiting {
                    server.waiting = false
                    server.mutex.Unlock()
                    last_run = true
                    continue
                } else {
                    server.mutex.Unlock()
                    return
                }
            } else {
                continue
            }
        }
        if wstatus.Exited() {
            process := server.lookup(pid)
            if process != nil {
                process.setExitcode(wstatus.ExitStatus())
            }
        }
        if last_run {
            break
        }
    }
}

func Run(file *os.File) {

    // Create our server.
    server := new(Server)
    server.active = make(map[int]*Process)

    // Start our periodic clearer.
    server.clearPeriodic()

    // Create our RPC server.
    codec := jsonrpc.NewServerCodec(file)
    rpcserver := rpc.NewServer()
    rpcserver.Register(server)

    // Listen for children.
    go server.wait()

    // Service requests.
    rpcserver.ServeCodec(codec)
}
