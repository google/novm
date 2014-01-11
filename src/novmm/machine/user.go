package machine

import (
    "novmm/platform"
    "sort"
    "syscall"
)

type UserMemory struct {
    BaseDevice

    // As laid-out.
    // This is indexed by offset in the file,
    // and each offset points to the given region.
    Map map[uint64]MemoryRegion `json:"map"`

    // The offset in the file.
    Offset int64 `json:"offset"`

    // The FD to map for regions.
    Fd  int `json:"fd"`

    // Our map.
    mmap []byte
}

func (user *UserMemory) Reload(
    vm *platform.Vm,
    model *Model) (uint64, uint64, error) {

    total := uint64(0)
    max_offset := uint64(0)

    // Place existing user memory.
    for offset, region := range user.Map {

        // Allocate it in the machine.
        err := model.Reserve(
            vm,
            user,
            MemoryTypeUser,
            region.Start,
            region.Size,
            user.mmap[offset:offset+region.Size])
        if err != nil {
            return total, max_offset, err
        }

        // Is our max_offset up to date?
        if offset+region.Size > max_offset {
            max_offset = offset + region.Size
        }

        // Done some more.
        total += region.Size
    }

    return total, max_offset, nil
}

func (user *UserMemory) Layout(
    vm *platform.Vm,
    model *Model,
    start uint64,
    memory uint64) error {

    // Try to place our user memory.
    // NOTE: This will be called after all devices
    // have reserved appropriate memory regions, so
    // we will not conflict with anything else.
    last_top := platform.Paddr(0)

    sort.Sort(&model.MemoryMap)

    for i := 0; i < len(model.MemoryMap) && memory > 0; i += 1 {

        region := model.MemoryMap[i]

        if last_top != region.Start {

            // How much can we do here?
            gap := uint64(region.Start) - uint64(last_top)
            if gap > memory {
                gap = memory
                memory = 0
            } else {
                memory -= gap
            }

            // Allocate the bits.
            err := model.Reserve(
                vm,
                user,
                MemoryTypeUser,
                last_top,
                gap,
                user.mmap[start:start+gap])
            if err != nil {
                return err
            }

            // Move ahead in the backing store.
            start += gap
        }

        // Remember the top of this region.
        last_top = region.Start.After(region.Size)
    }

    if memory > 0 {
        err := model.Reserve(
            vm,
            user,
            MemoryTypeUser,
            last_top,
            memory,
            user.mmap[start:])
        if err != nil {
            return err
        }
    }

    // All is good.
    return nil
}

func NewUserMemory(info *DeviceInfo) (Device, error) {

    // Create our user memory.
    // Nothing special, no defaults.
    user := new(UserMemory)
    return user, user.Init(info)
}

func (user *UserMemory) Attach(vm *platform.Vm, model *Model) error {

    // Create a mmap'ed region.
    var stat syscall.Stat_t
    err := syscall.Fstat(user.Fd, &stat)
    if err != nil {
        return err
    }

    // How big is our memory?
    size := uint64(stat.Size)

    if size > 0 {
        user.mmap, err = syscall.Mmap(
            user.Fd,
            user.Offset,
            int(size),
            syscall.PROT_READ|syscall.PROT_WRITE,
            syscall.MAP_SHARED)
        if err != nil || user.mmap == nil {
            return err
        }
    } else {
        user.mmap = make([]byte, 0)
    }

    // Layout the existing regions.
    total, max_offset, err := user.Reload(vm, model)
    if err != nil {
        return err
    }

    // Layout remaining amount.
    if size > total {
        return user.Layout(vm, model, max_offset, size-total)
    } else {
        return nil
    }
}
