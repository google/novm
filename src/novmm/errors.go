package main

import (
    "errors"
)

var ExitWithoutReason = errors.New("Exit without reason?")
var NoVcpus = errors.New("No vcpus?")
var InvalidControlSocket = errors.New("Invalid control socket?")
