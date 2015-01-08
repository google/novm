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
#include "kvm_exits.h"
#include <string.h>
*/
import "C"

import (
	"unsafe"
)

//export kvmExitMmio
func kvmExitMmio(
	addr C.__u64,
	data *C.__u64,
	length C.__u32,
	write C.int) unsafe.Pointer {

	return unsafe.Pointer(&ExitMmio{
		addr:   Paddr(addr),
		data:   (*uint64)(data),
		length: uint32(length),
		write:  write != C.int(0),
	})
}

//export kvmExitPio
func kvmExitPio(
	port C.__u16,
	size C.__u8,
	data unsafe.Pointer,
	length C.__u32,
	out C.int) unsafe.Pointer {

	return unsafe.Pointer(&ExitPio{
		port: Paddr(port),
		size: uint8(size),
		data: (*uint64)(data),
		out:  out != C.int(0),
	})
}

//export kvmExitInternalError
func kvmExitInternalError(
	code C.__u32) unsafe.Pointer {

	return unsafe.Pointer(&ExitInternalError{
		code: uint32(code),
	})
}

//export kvmExitException
func kvmExitException(
	exception C.__u32,
	error_code C.__u32) unsafe.Pointer {

	return unsafe.Pointer(&ExitException{
		exception: uint32(exception),
		errorCode: uint32(error_code),
	})
}

//export kvmExitUnknown
func kvmExitUnknown(
	code C.__u32) unsafe.Pointer {

	return unsafe.Pointer(&ExitUnknown{
		code: uint32(code),
	})
}

func (vcpu *Vcpu) GetExitError() error {
	// Handle the error.
	switch C.int(vcpu.kvm.exit_reason) {
	case C.ExitReasonMmio:
		return (*ExitMmio)(C.handle_exit_mmio(vcpu.kvm))
	case C.ExitReasonIo:
		return (*ExitPio)(C.handle_exit_io(vcpu.kvm))
	case C.ExitReasonInternalError:
		return (*ExitInternalError)(C.handle_exit_internal_error(vcpu.kvm))
	case C.ExitReasonException:
		return (*ExitException)(C.handle_exit_exception(vcpu.kvm))
	case C.ExitReasonDebug:
		return &ExitDebug{}
	case C.ExitReasonShutdown:
		return &ExitShutdown{}
	default:
		return (*ExitUnknown)(C.handle_exit_unknown(vcpu.kvm))
	}

	// Unreachable.
	return nil
}
