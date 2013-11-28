package machine

import (
    "log"
    "novmm/platform"
    "sort"
    "unsafe"
)

type MemoryType int

const (
    Reserved MemoryType = iota
    User                = 1
    Acpi                = 2
    Special             = 3
)

type MemoryRegion struct {
    Start platform.Paddr
    Size  uint64
}

type TypedMemoryRegion struct {
    MemoryRegion
    Type MemoryType

    // The mmap for user memory.
    // NOTE: If the user passed in a user pointer,
    // we will store it directly here and use it.
    // If they pass in a mmap, we will ensure that
    // the user pointer is also converted properly.
    mmap []byte
    user unsafe.Pointer

    // Whether the above was allocated?
    allocated bool
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

type MemoryMap struct {
    // The list of memory regions.
    regions []*TypedMemoryRegion

    // The VM in which they are created.
    vm  platform.Vm
}

func (memory *MemoryMap) Len() int {
    return len(memory.regions)
}

func (memory *MemoryMap) Swap(i int, j int) {
    memory.regions[i], memory.regions[j] = memory.regions[j], memory.regions[i]
}

func (memory *MemoryMap) Less(i int, j int) bool {
    return memory.regions[i].Start < memory.regions[j].Start
}

func (memory *MemoryMap) HasConflict(start platform.Paddr, size uint64) bool {
    for _, orig_region := range memory.regions {
        if orig_region.Overlaps(start, size) {
            return true
        }
    }

    return false
}

func (memory *MemoryMap) add(region *TypedMemoryRegion) error {
    if memory.HasConflict(region.Start, region.Size) {
        return MemoryConflict
    }
    memory.regions = append(memory.regions, region)
    sort.Sort(memory)
    return nil
}

func (memory *MemoryMap) remove(region *TypedMemoryRegion) error {
    for i, orig_region := range memory.regions {
        if orig_region.Start == region.Start &&
            orig_region.Size == region.Size {

            lost_region := memory.regions[i]

            if i != len(memory.regions)-1 {
                // Move the last element to this position.
                memory.regions[i] = memory.regions[len(memory.regions)-1]
            }
            memory.regions = memory.regions[0 : len(memory.regions)-1]
            sort.Sort(memory)

            // Free memory if necessary.
            if lost_region.allocated {
                memory.vm.DeleteUserMemory(lost_region.mmap)
            }

            return nil
        }
    }
    return MemoryNotFound
}

func (memory *MemoryMap) Max() platform.Paddr {

    if len(memory.regions) == 0 {
        // No memory available?
        return platform.Paddr(0)
    }

    // Return the highest available address.
    top := memory.regions[len(memory.regions)-1]
    return top.End()
}

func (memory *MemoryMap) Allocate(
    memtype MemoryType,
    name string,
    start platform.Paddr,
    size uint64,
    end platform.Paddr,
    alignment uint) ([]byte, platform.Paddr, error) {

    // Make sure the size is page-aligned.
    size = platform.Align(size, platform.PageSize, true)

    // Allocate new user memory.
    var mmap []byte
    var user unsafe.Pointer
    if memtype == User || memtype == Acpi {
        var err error
        mmap, err = memory.vm.CreateUserMemory(size)
        if err != nil {
            return nil, platform.Paddr(0), err
        }
        user = unsafe.Pointer(&mmap[0])
    }

    region, err := memory.reserve(
        memtype,
        name,
        start,
        size,
        end,
        alignment,
        mmap,
        user,
        mmap != nil)
    if err != nil {
        if mmap != nil {
            memory.vm.DeleteUserMemory(mmap)
        }
        return nil, region.Start, nil
    }

    // We're good.
    return mmap, region.Start, nil
}

func (memory *MemoryMap) Load(
    memtype MemoryType,
    name string,
    start platform.Paddr,
    mmap []byte,
    alignment uint) (platform.Paddr, error) {

    // Make sure the size is page aligned.
    size := platform.Align(uint64(len(mmap)), platform.PageSize, true)

    region, err := memory.reserve(
        memtype,
        name,
        start,
        size,
        memory.Max().Align(alignment, true),
        alignment,
        mmap,
        unsafe.Pointer(&mmap[0]),
        false)
    if err != nil {
        return region.Start, err
    }

    // We're good.
    return region.Start, err
}

func (memory *MemoryMap) Set(
    memtype MemoryType,
    name string,
    start platform.Paddr,
    size uint64,
    user unsafe.Pointer) error {

    _, err := memory.reserve(
        memtype,
        name,
        start,
        size,
        start,
        platform.PageSize,
        nil,
        user,
        false)

    return err
}

func (memory *MemoryMap) reserve(
    memtype MemoryType,
    name string,
    start platform.Paddr,
    size uint64,
    max platform.Paddr,
    alignment uint,
    mmap []byte,
    user unsafe.Pointer,
    allocated bool) (MemoryRegion, error) {

    // Ensure all targets are aligned.
    if (start.Align(platform.PageSize, false) != start) ||
        (size%platform.PageSize != 0) {
        return MemoryRegion{}, MemoryUnaligned
    }

    // Verbose messages.
    log.Printf(
        "memory: reserving (type: %d, align: %x) of size %x in [%x,%x) for %s...",
        memtype,
        alignment,
        size,
        start,
        max.After(size),
        name)

    var search platform.Paddr

    for search = start.Align(alignment, true); search <= max; search = search.After(uint64(alignment)) {

        if !memory.HasConflict(search, size) {
            region := &TypedMemoryRegion{
                MemoryRegion: MemoryRegion{search, size},
                Type:         memtype,
                mmap:         mmap,
                user:         user,
                allocated:    allocated,
            }
            err := memory.add(region)
            return region.MemoryRegion, err
        }
    }

    // Beyond our current max?
    if search > max {
        region := &TypedMemoryRegion{
            MemoryRegion: MemoryRegion{search, size},
            Type:         memtype,
            mmap:         mmap,
            user:         user,
            allocated:    allocated,
        }
        err := memory.add(region)
        return region.MemoryRegion, err
    }

    // Nothing found.
    return MemoryRegion{}, MemoryNotFound
}

func (memory *MemoryMap) Map() error {

    // Setup each region.
    for _, region := range memory.regions {

        var err error

        switch region.Type {
        case User:
            err = memory.vm.MapUserMemory(region.Start, region.Size, region.user)
        case Reserved:
            err = memory.vm.MapReservedMemory(region.Start, region.Size)
        case Acpi:
            err = memory.vm.MapUserMemory(region.Start, region.Size, region.user)
        case Special:
            region.Size, err = memory.vm.MapSpecialMemory(region.Start)
        }

        if err != nil {
            memory.Unmap()
            return err
        }
    }

    // We're good.
    return nil
}

func (memory *MemoryMap) Unmap() error {

    // Teardown each region.
    for _, region := range memory.regions {

        switch region.Type {
        case User:
            memory.vm.UnmapUserMemory(region.Start, region.Size)

        case Reserved:
            memory.vm.UnmapReservedMemory(region.Start, region.Size)

        case Acpi:
            memory.vm.UnmapUserMemory(region.Start, region.Size)

        case Special:
            memory.vm.UnmapSpecialMemory(region.Start)
        }
    }

    // Always succeed.
    return nil
}
