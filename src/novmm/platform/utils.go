package platform

import (
    "unsafe"
)

func AlignBytes(data []byte) []byte {
    if uintptr(unsafe.Pointer(&data[0]))%PageSize != 0 {
        orig_data := data
        data = make([]byte, len(orig_data), len(orig_data))
        copy(data, orig_data)
    }
    return data
}
