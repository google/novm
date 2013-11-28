package main

import (
    "errors"
)

var ExitWithoutReason = errors.New("Exit without reason?")
var NoAcpiDataProvided = errors.New("No ACPI data provided?")
var NoKernelProvided = errors.New("No kernel provided?")
