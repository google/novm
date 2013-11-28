package platform

// Basic abstractions.
type Irq uint32

// Address types.
type Vaddr uint64
type Paddr uint64

func Align(addr uint64, alignment uint, up bool) uint64 {

    // Aligned already?
    if addr%uint64(alignment) == 0 {
        return addr
    }

    // Give the closest aligned address.
    addr = addr - (addr % uint64(alignment))

    if up {
        // Should we align up?
        return addr + uint64(alignment)
    }
    return addr
}

func (paddr Paddr) Align(alignment uint, up bool) Paddr {
    return Paddr(Align(uint64(paddr), alignment, up))
}

func (paddr Paddr) OffsetFrom(base Paddr) uint64 {
    return uint64(paddr) - uint64(base)
}

func (paddr Paddr) After(length uint64) Paddr {
    return Paddr(uint64(paddr) + uint64(length))
}
