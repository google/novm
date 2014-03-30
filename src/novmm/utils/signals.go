package utils

import (
    "syscall"
)

const (
    SigVcpuInt        = syscall.SIGUSR1
    SigShutdown       = syscall.SIGTERM
    SigRestart        = syscall.SIGHUP
    SigSpecialRestart = syscall.SIGUSR2
)
