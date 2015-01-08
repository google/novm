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

// +build linux
package platform

/*
#include <sys/eventfd.h>

const int EfdCloExec = EFD_CLOEXEC;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

// Event server.
//
// This file was created in the hopes that I would
// be able to bolt on an event server to the internal
// network hub. Not so simple. That's all in the net
// namespace, and very much network-specific.
//
// So... for now, this will just use blocking system
// calls. It's relatively lightweight and we're not scaling
// to thousands of concurrent goroutines, just dozens.
//
// In the future, this is a great opportunity to improve
// the core archiecture (and eliminate a few system threads).

type EventFd struct {
	// Underlying FD.
	// NOTE: In reality we may want to serialize read/write
	// access to this fd as I'm fairly sure we will end up
	// with unexpected errors if this interface is used in
	// this way. However, we'll keep this as a simple runtime
	// adaptor and punt that complexity up to the next level.
	fd int
}

func NewEventFd() (*EventFd, error) {
	// Create new eventfd.
	// NOTE: It's critical that it's non-blocking for the hub
	// integration below (otherwise it'll just end up blocking
	// in the Read() or Write() system call.
	// But given that we aren't using the hub, for now this is
	// just a regular blocking call. C'est la vie.
	fd, _, e := syscall.Syscall(
		syscall.SYS_EVENTFD,
		0,
		uintptr(C.EfdCloExec),
		0)
	if e != 0 {
		return nil, syscall.Errno(e)
	}

	eventfd := &EventFd{fd: int(fd)}
	return eventfd, nil
}

func (fd *EventFd) Close() error {
	return syscall.Close(fd.fd)
}

func (fd *EventFd) Fd() int {
	return fd.fd
}

func (fd *EventFd) Wait() (uint64, error) {
	for {
		var val uint64
		_, _, err := syscall.Syscall(
			syscall.SYS_READ,
			uintptr(fd.fd),
			uintptr(unsafe.Pointer(&val)),
			8)
		if err != 0 {
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			return 0, err
		}
		return val, nil
	}

	// Unreachable.
	return 0, nil
}

func (fd *EventFd) Signal(val uint64) error {
	for {
		var val uint64
		_, _, err := syscall.Syscall(
			syscall.SYS_WRITE,
			uintptr(fd.fd),
			uintptr(unsafe.Pointer(&val)),
			8)
		if err != 0 {
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			return err
		}
		return nil
	}

	// Unreachable.
	return nil
}
