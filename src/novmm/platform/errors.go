package platform

import (
    "errors"
)

// Serialization.
var VcpuIncompatible = errors.New("Incompatible VCPU data?")
var PitIncompatible = errors.New("Incompatible PIT state?")
var IrqChipIncompatible = errors.New("Incompatible IRQ chip state?")
var LApicIncompatible = errors.New("Incompatible LApic state?")

// Register errors.
var UnknownRegister = errors.New("Unknown Register")

// Vcpu state errors.
var NotPaused = errors.New("Vcpu is not paused?")
var AlreadyPaused = errors.New("Vcpu is already paused.")
var UnknownState = errors.New("Unknown vcpu state?")
