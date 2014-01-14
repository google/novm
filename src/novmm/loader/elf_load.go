package loader

/*
#include <string.h>
#include "elf_load.h"
*/
import "C"

import (
    "log"
    "novmm/machine"
    "novmm/platform"
    "syscall"
    "unsafe"
)

//export doLoad
func doLoad(
    self unsafe.Pointer,
    offset C.size_t,
    source unsafe.Pointer,
    length C.size_t) C.int {

    model := (*machine.Model)(self)

    // Bump up the size to the end of the page.
    new_length := platform.Align(uint64(length), platform.PageSize, true)

    // Allocate the backing data.
    data, err := model.Map(
        machine.MemoryTypeUser,
        platform.Paddr(offset),
        new_length,
        true)
    if err != nil {
        // Things are broken.
        log.Print("Error during ElfLoad: ", err)
        return -C.int(syscall.EINVAL)
    }

    // Copy the data in.
    C.memcpy(unsafe.Pointer(&data[0]), source, length)

    // All good.
    return 0
}

func ElfLoad(
    data []byte,
    model *machine.Model) (uint64, bool, error) {

    // Do the load.
    var is_64bit C.int
    entry_point := C.elf_load(
        unsafe.Pointer(&data[0]),
        unsafe.Pointer(model),
        &is_64bit)
    if entry_point < 0 {
        return 0, false, syscall.Errno(-entry_point)
    }

    // Looks like we're okay.
    return uint64(entry_point), int(is_64bit) == 1, nil
}
