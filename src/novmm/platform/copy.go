package platform

/*
#include <string.h>
*/
import "C"

import (
    "unsafe"
)

func Copy(
    mmap []byte,
    mmap_offset uint64,
    data []byte) error {

    return CopyUnsafe(
        mmap,
        mmap_offset,
        unsafe.Pointer(&data[0]),
        uint64(len(data)))
}

func CopyUnsafe(
    mmap []byte,
    mmap_offset uint64,
    data unsafe.Pointer,
    length uint64) error {

    // Assume success.
    C.memcpy(
        unsafe.Pointer(&mmap[mmap_offset]),
        data,
        C.size_t(length))
    return nil
}
