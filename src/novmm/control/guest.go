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

package control

import (
	"net/rpc"
	"net/rpc/jsonrpc"
	"noguest/protocol"
)

func (control *Control) init() {

	buffer := make([]byte, 1, 1)

	// Read our control byte back.
	n, err := control.proxy.Read(buffer)
	if n == 1 && err == nil {
		switch buffer[0] {
		case protocol.NoGuestStatusOkay:
			break
		case protocol.NoGuestStatusFailed:
			// Something went horribly wrong.
			control.client_res <- InternalGuestError
			return
		default:
			// This isn't good, who knows what happened?
			control.client_res <- protocol.UnknownStatus
			return
		}
	} else if err != nil {
		// An actual error.
		control.client_res <- err
		return
	}

	// Send our control byte to noguest.
	// This essentially controls how the guest
	// will proceed during execution. If it is the
	// real init process, it will wait for run commands
	// and execute the given processes inside the VM.
	// If it is not the real init process, it will fork
	// and execute the real init before starting to
	// process any other RPC commands.
	if control.real_init {
		buffer[0] = protocol.NoGuestCommandRealInit
	} else {
		buffer[0] = protocol.NoGuestCommandFakeInit
	}
	n, err = control.proxy.Write(buffer)
	if n != 1 {
		// Can't send anything?
		control.client_res <- InternalGuestError
		return
	}

	// Looks like we're good.
	control.client_res <- nil
}

func (control *Control) barrier() {
	control.client_err = <-control.client_res
	control.client_codec = jsonrpc.NewClientCodec(control.proxy)
	control.client = rpc.NewClientWithCodec(control.client_codec)
}

func (control *Control) Ready() (*rpc.Client, error) {
	control.client_once.Do(control.barrier)
	return control.client, control.client_err
}
