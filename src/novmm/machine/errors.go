package machine

import (
    "errors"
    "fmt"
)

// Memory allocation / layout errors.
var MemoryConflict = errors.New("Memory regions conflict!")
var MemoryNotFound = errors.New("Memory region not found!")
var MemoryBusy = errors.New("Memory could not be allocated!")
var MemoryUnaligned = errors.New("Memory not aligned!")
var UserMemoryNotFound = errors.New("No user memory found?")

// Interrupt allocation errors.
var InterruptConflict = errors.New("Device interrupt conflict!")
var InterruptUnavailable = errors.New("No interrupt available!")

// PCI errors.
var PciInvalidAddress = errors.New("Invalid PCI address!")
var PciBusNotFound = errors.New("Requested PCI devices, but no bus found?")

// UART errors.
var UartUnknown = errors.New("Unknown COM port.")

// Driver errors.
func DriverUnknown(name string) error {
    return errors.New(fmt.Sprintf("Unknown driver: %s", name))
}
