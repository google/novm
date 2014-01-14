package machine

import (
    "log"
    "novmm/platform"
    "sort"
)

type MemoryType int

const (
    MemoryTypeReserved MemoryType = iota
    MemoryTypeUser                = 1
    MemoryTypeAcpi                = 2
    MemoryTypeSpecial             = 3
)

type MemoryRegion struct {
    Start platform.Paddr
    Size  uint64
}

type TypedMemoryRegion struct {
    MemoryRegion
    MemoryType

    // The owner.
    Device

    // The memory pointer (slice).
    user []byte

    // Allocated chunks.
    // These are offsets, which point
    // to the amount of memory allocated.
    allocated map[uint64]uint64
}

func (region *MemoryRegion) End() platform.Paddr {
    return region.Start.After(region.Size)
}

func (region *MemoryRegion) Overlaps(start platform.Paddr, size uint64) bool {
    return ((region.Start >= start && region.Start < start.After(size)) ||
        (region.End() > start && region.End() <= start.After(size)))
}

func (region *MemoryRegion) Contains(start platform.Paddr, size uint64) bool {
    return region.Start <= start && region.End() >= start.After(size)
}

type MemoryMap []*TypedMemoryRegion

func (memory *MemoryMap) Len() int {
    return len(*memory)
}

func (memory *MemoryMap) Swap(i int, j int) {
    (*memory)[i], (*memory)[j] = (*memory)[j], (*memory)[i]
}

func (memory *MemoryMap) Less(i int, j int) bool {
    return (*memory)[i].Start < (*memory)[j].Start
}

func (memory *MemoryMap) Conflicts(start platform.Paddr, size uint64) bool {
    for _, orig_region := range *memory {
        if orig_region.Overlaps(start, size) {
            return true
        }
    }
    return false
}

func (memory *MemoryMap) Add(region *TypedMemoryRegion) error {
    if memory.Conflicts(region.Start, region.Size) {
        return MemoryConflict
    }

    *memory = append(*memory, region)
    sort.Sort(memory)
    return nil
}

func (memory *MemoryMap) Max() platform.Paddr {
    if len(*memory) == 0 {
        // No memory available?
        return platform.Paddr(0)
    }

    // Return the highest available address.
    top := (*memory)[len(*memory)-1]
    return top.End()
}

func (memory *MemoryMap) Reserve(
    vm *platform.Vm,
    device Device,
    memtype MemoryType,
    start platform.Paddr,
    size uint64,
    user []byte) error {

    _, err := memory.Select(
        vm,
        device,
        memtype,
        start,
        size,
        start,
        user)
    return err
}

func (memory *MemoryMap) ReserveFind(
    vm *platform.Vm,
    device Device,
    memtype MemoryType,
    start platform.Paddr,
    size uint64,
    max platform.Paddr,
    user []byte) (MemoryRegion, error) {

    region, err := memory.Select(
        vm,
        device,
        memtype,
        start,
        size,
        max,
        user)
    if err != nil {
        return region, err
    }

    return region, err
}

func (memory *MemoryMap) Select(
    vm *platform.Vm,
    device Device,
    memtype MemoryType,
    start platform.Paddr,
    size uint64,
    max platform.Paddr,
    user []byte) (MemoryRegion, error) {

    // Ensure all targets are aligned.
    if (start.Align(platform.PageSize, false) != start) ||
        (size%platform.PageSize != 0) {
        return MemoryRegion{}, MemoryUnaligned
    }

    // Verbose messages.
    if device.IsDebugging() {
        log.Printf(
            "model: %s: reserving (type: %d) of size %x in [%x,%x]",
            device.Name(),
            memtype,
            size,
            start,
            max.After(size-1))
    }

    search := start
    var region *TypedMemoryRegion

    // Can we find a region?
    for search <= max {
        if !memory.Conflicts(search, size) {
            region = &TypedMemoryRegion{
                MemoryRegion: MemoryRegion{search, size},
                MemoryType:   memtype,
                Device:       device,
                user:         user,
                allocated:    make(map[uint64]uint64),
            }
            err := memory.Add(region)
            if err != nil {
                return region.MemoryRegion, err
            }
        }
        search += platform.PageSize
    }

    // Beyond our current max?
    if search > memory.Max() && start <= memory.Max() {
        region = &TypedMemoryRegion{
            MemoryRegion: MemoryRegion{memory.Max(), size},
            MemoryType:   memtype,
            Device:       device,
            user:         user,
            allocated:    make(map[uint64]uint64),
        }
        err := memory.Add(region)
        if err != nil {
            return region.MemoryRegion, err
        }
    }

    // Nothing found.
    if region == nil {
        log.Printf("model: conflict for %s: %x bytes in [%x,%x].",
            device.Name(),
            size,
            start,
            max.After(size))
        return MemoryRegion{}, MemoryConflict
    }

    // Do the mapping.
    var err error
    switch region.MemoryType {
    case MemoryTypeUser:
        err = vm.MapUserMemory(region.Start, region.Size, region.user)
    case MemoryTypeReserved:
        err = vm.MapReservedMemory(region.Start, region.Size)
    case MemoryTypeAcpi:
        err = vm.MapUserMemory(region.Start, region.Size, region.user)
    case MemoryTypeSpecial:
        err = vm.MapSpecialMemory(region.Start)
    }

    // We're good?
    return region.MemoryRegion, err
}

func (memory *MemoryMap) Map(
    addr platform.Paddr,
    size uint64,
    allocate bool) ([]byte, error) {

    for i := 0; i < len(*memory); i += 1 {

        region := (*memory)[i]

        if region.Contains(addr, size) &&
            region.MemoryType == MemoryTypeUser {

            addr_offset := uint64(addr - region.Start)

            if allocate {
                // Mark it as used.
                for offset, alloc_size := range region.allocated {
                    if (addr_offset >= offset &&
                        addr_offset < offset+alloc_size) ||
                        (addr_offset+size >= offset &&
                            addr_offset < offset) {

                        // Already allocated?
                        return nil, MemoryConflict
                    }
                }

                // Found it.
                region.allocated[addr_offset] = size
            }

            return region.user[addr_offset : addr_offset+size], nil
        }
    }

    return nil, MemoryNotFound
}

func (memory *MemoryMap) Allocate(
    start platform.Paddr,
    end platform.Paddr,
    size uint64,
    top bool) (platform.Paddr, []byte, error) {

    if top {
        for ; end >= start; end -= platform.PageSize {

            mmap, _ := memory.Map(end, size, true)
            if mmap != nil {
                return end, mmap, nil
            }
        }

    } else {
        for ; start <= end; start += platform.PageSize {

            mmap, _ := memory.Map(start, size, true)
            if mmap != nil {
                return start, mmap, nil
            }
        }
    }

    // Couldn't find available memory.
    return platform.Paddr(0), nil, MemoryNotFound
}

func (memory *MemoryMap) Load(
    start platform.Paddr,
    end platform.Paddr,
    data []byte,
    top bool) (platform.Paddr, error) {

    // Allocate the backing data.
    addr, backing_mmap, err := memory.Allocate(
        start,
        end,
        uint64(len(data)),
        top)
    if err != nil {
        return platform.Paddr(0), err
    }

    // Copy it in.
    copy(backing_mmap, data)

    // We're good.
    return addr, nil
}
