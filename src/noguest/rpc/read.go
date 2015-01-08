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

type ReadCommand struct {

	// The relevant pid.
	Pid int `json:"pid"`

	// How much to read?
	N uint `json:"n"`
}

type ReadResult struct {

	// The data read.
	Data []byte `json:"data"`
}

func (server *Server) Read(
	read *ReadCommand,
	result *ReadResult) error {

	process := server.lookup(read.Pid)
	if process == nil {
		result.Data = []byte{}
		return nil
	}

	// Read available data.
	buffer := make([]byte, read.N, read.N)
	n, err := process.output.Read(buffer)
	if n > 0 {
		result.Data = buffer[:n]
	} else {
		result.Data = []byte{}
	}

	return err
}
