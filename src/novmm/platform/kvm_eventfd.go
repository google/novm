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
#include <linux/kvm.h>

// IOCTL calls.
const int IoctlIoEventFd = KVM_IOEVENTFD;

// IOCTL flags.
const int IoctlIoEventFdFlagPio = KVM_IOEVENTFD_FLAG_PIO;
const int IoctlIoEventFdFlagDatamatch = KVM_IOEVENTFD_FLAG_DATAMATCH;
const int IoctlIoEventFdFlagDeassign = KVM_IOEVENTFD_FLAG_DEASSIGN;
*/
import "C"

import (
	"syscall"
	"unsafe"
)

type BoundEventFd struct {

	// Our system eventfd.
	*EventFd

	// Our VM reference.
	*Vm

	// Address information.
	paddr  Paddr
	size   uint
	is_pio bool

	// Value information.
	has_value bool
	value     uint64
}

func (vm *Vm) SetEventFd(
	eventfd *EventFd,
	paddr Paddr,
	size uint,
	is_pio bool,
	unbind bool,
	has_value bool,
	value uint64) error {

	var ioeventfd C.struct_kvm_ioeventfd
	ioeventfd.addr = C.__u64(paddr)
	ioeventfd.len = C.__u32(size)
	ioeventfd.fd = C.__s32(eventfd.Fd())
	ioeventfd.datamatch = C.__u64(value)

	if is_pio {
		ioeventfd.flags |= C.__u32(C.IoctlIoEventFdFlagPio)
	}
	if unbind {
		ioeventfd.flags |= C.__u32(C.IoctlIoEventFdFlagDeassign)
	}
	if has_value {
		ioeventfd.flags |= C.__u32(C.IoctlIoEventFdFlagDatamatch)
	}

	// Bind / unbind the eventfd.
	_, _, e := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(vm.fd),
		uintptr(C.IoctlIoEventFd),
		uintptr(unsafe.Pointer(&ioeventfd)))
	if e != 0 {
		return e
	}

	// Success.
	return nil
}

func (vm *Vm) NewBoundEventFd(
	paddr Paddr,
	size uint,
	is_pio bool,
	has_value bool,
	value uint64) (*BoundEventFd, error) {

	// Create our system eventfd.
	eventfd, err := NewEventFd()
	if err != nil {
		return nil, err
	}

	// Bind the eventfd.
	err = vm.SetEventFd(
		eventfd,
		paddr,
		size,
		is_pio,
		false,
		has_value,
		value)
	if err != nil {
		eventfd.Close()
		return nil, err
	}

	// Return our bound event.
	return &BoundEventFd{
		EventFd:   eventfd,
		Vm:        vm,
		paddr:     paddr,
		size:      size,
		is_pio:    is_pio,
		has_value: has_value,
		value:     value,
	}, nil
}

func (fd *BoundEventFd) Close() error {

	// Unbind the event.
	err := fd.Vm.SetEventFd(
		fd.EventFd,
		fd.paddr,
		fd.size,
		fd.is_pio,
		true,
		fd.has_value,
		fd.value)
	if err != nil {
		return err
	}

	// Close the eventfd.
	return fd.Close()
}
