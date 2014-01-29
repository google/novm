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
    "path"
    "syscall"
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
        var stat syscall.Stat_t
        err := syscall.Stat(
            path.Join(filepath, ".deleted"),
            &stat)
        if err != nil {
            return false, err
        }
        return true, nil
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
        fd, err := syscall.Open(
            path.Join(filepath, ".deleted"),
            syscall.O_RDWR|syscall.O_CREAT,
            syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR)
        if err == nil {
            syscall.Close(fd)
        }
        return err
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
        return syscall.Unlink(path.Join(filepath, ".deleted"))
    }
    return nil
}
