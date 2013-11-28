package platform

import (
    "errors"
)

var InvalidVcpus = errors.New("Invalid number of VCPUs")
var InvalidMemory = errors.New("Invalid memory size")
