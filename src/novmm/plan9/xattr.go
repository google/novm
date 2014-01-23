// Copyright 2009 The Go9p Authors.  All rights reserved.
// Copyright 2013 Adin Scannell.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the licenses/go9p file.
package plan9

/*
#include <errno.h>
#include <sys/types.h>
#include <sys/xattr.h>
*/
import "C"

import (
    "unsafe"
)

func readdelattr(filepath string) (bool, error) {
    val := C.char(0)
    err := C.getxattr(
        C.CString(filepath),
        C.CString("novm-deleted"),
        unsafe.Pointer(&val),
        C.size_t(1))
    if err != C.ssize_t(1) {
        return false, XattrError
    }
    return val == C.char(1), nil
}

func setdelattr(filepath string) error {
    val := C.char(1)
    e := C.setxattr(
        C.CString(filepath),
        C.CString("novm-deleted"),
        unsafe.Pointer(&val),
        C.size_t(1),
        C.int(0))
    if e != 0 {
        return XattrError
    }
    return nil
}

func cleardelattr(filepath string) error {
    val := C.char(0)
    e := C.setxattr(
        C.CString(filepath),
        C.CString("novm-deleted"),
        unsafe.Pointer(&val),
        C.size_t(1),
        C.int(0))
    if e != 0 {
        return XattrError
    }
    return nil
}
