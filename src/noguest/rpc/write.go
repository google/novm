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

package rpc

type WriteCommand struct {

	// The relevant pid.
	Pid int `json:"pid"`

	// The write.
	Data []byte `json:"data"`
}

type WriteResult struct {

	// How much was written?
	Written int `json:"n"`
}

func (server *Server) Write(
	write *WriteCommand,
	out *WriteResult) error {

	process := server.lookup(write.Pid)
	if process == nil || write.Data == nil {
		out.Written = -1
		return nil
	}

	// Push the write.
	for len(write.Data) > 0 {
		n, err := process.input.Write(write.Data)
		out.Written += n
		write.Data = write.Data[n:]
		if err != nil {
			return err
		}
	}

	return nil
}
