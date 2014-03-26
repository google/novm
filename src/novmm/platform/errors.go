package platform

import (
    "errors"
)

// Serialization.
var VcpuIncompatible = errors.New("Incompatible VCPU data?")

// Vcpu state errors.
var NotPaused = errors.New("Vcpu is not paused?")
var AlreadyPaused = errors.New("Vcpu is already paused.")
