package machine

import (
    "errors"
    "fmt"
)

// Memory allocation / layout errors.
var MemoryConflict = errors.New("Memory regions conflict!")
var MemoryNotFound = errors.New("Memory region not found!")
var MemoryUnaligned = errors.New("Memory not aligned!")

// PCI errors.
var PciInvalidAddress = errors.New("Invalid PCI address!")

// UART errors.
var UartUnknown = errors.New("Unknown COM port.")

// Driver errors.
func DriverUnknown(name string) error {
    return errors.New(fmt.Sprintf("Unknown driver: %s", name))
}

// Virtio errors.
var VirtioInvalidRegister = errors.New("Invalid virtio register?")
var VirtioPciNotFound = errors.New("Requested PCI devices, but no bus found?")
