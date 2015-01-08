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

package platform

import (
	"fmt"
)

// Exit events.
// When running the vcpu (i.e. vcpu.Run()), you should check
// the return value to see if it matches these types. They should
// be handled appropriate and Run() can be called again to resume.
type ExitUnknown struct {
	code uint32
}

func (exit *ExitUnknown) Error() string {
	return fmt.Sprintf("Unknown exit reason: %d", exit.code)
}

type ExitMmio struct {
	addr   Paddr
	data   *uint64
	length uint32
	write  bool
}

func (exitmmio *ExitMmio) Addr() Paddr {
	return exitmmio.addr
}

func (exitmmio *ExitMmio) Data() *uint64 {
	return exitmmio.data
}

func (exitmmio *ExitMmio) Length() uint {
	return uint(exitmmio.length)
}

func (exitmmio *ExitMmio) IsWrite() bool {
	return exitmmio.write
}

func (exit *ExitMmio) Error() string {
	if exit.write {
		return fmt.Sprintf(
			"Memory-mapped write to %08x (length: %d)",
			exit.addr,
			exit.length)
	}
	return fmt.Sprintf(
		"Memory-mapped read from %08x (length: %d)",
		exit.addr,
		exit.length)
}

type ExitPio struct {
	port Paddr
	size uint8
	data *uint64
	out  bool
}

func (exitio *ExitPio) Port() Paddr {
	return exitio.port
}

func (exitio *ExitPio) Size() uint {
	return uint(exitio.size)
}

func (exitio *ExitPio) Data() *uint64 {
	return exitio.data
}

func (exitio *ExitPio) IsOut() bool {
	return exitio.out
}

func (exit *ExitPio) Error() string {
	if exit.out {
		return fmt.Sprintf(
			"Port out to %04x (size: %d)",
			exit.port,
			exit.size)
	}
	return fmt.Sprintf(
		"Port in from %04x (size: %d)",
		exit.port,
		exit.size)
}

type ExitInternalError struct {
	code uint32
}

func (exit *ExitInternalError) Error() string {
	return fmt.Sprintf("Internal error: %d", exit.code)
}

type ExitException struct {
	exception uint32
	errorCode uint32
}

func (exit *ExitException) Error() string {
	return fmt.Sprintf(
		"Exception (exception: %d, error_code: %d)",
		exit.exception,
		exit.errorCode)
}

type ExitDebug struct {
}

func (exit *ExitDebug) Error() string {
	return fmt.Sprintf("Debug exit (single-step)")
}

type ExitShutdown struct {
}

func (exit *ExitShutdown) Error() string {
	return fmt.Sprintf("Shutdown exit")
}
