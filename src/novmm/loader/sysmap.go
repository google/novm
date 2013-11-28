package loader

import (
    "novmm/platform"
)

type SystemMap interface {
    Lookup(addr platform.Vaddr) (string, uint64)
}

type Convention struct {
    instruction platform.Register
    arguments   []platform.Register
    rvalue      platform.Register
    stack       platform.Register
}
