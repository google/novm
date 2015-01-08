// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
