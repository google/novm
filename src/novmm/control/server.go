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
	noguest "noguest/rpc"
	"novmm/loader"
	"novmm/machine"
	"novmm/platform"
	"novmm/utils"
	"os"
	"sync"
	"syscall"
)

type Control struct {

	// The bound control fd.
	control_fd int

	// Should this instance use a real init?
	real_init bool

	// Our proxy to the in-guest agent.
	proxy machine.Proxy

	// Our rpc server.
	rpc *Rpc

	// Our bound client (to the in-guest agent).
	// NOTE: We have this setup as a lazy function
	// because the guest may take some small amount of
	// time before it's actually ready to process RPC
	// requests. We don't want this to interfere with
	// our ability to process our host-side requests.
	client_res   chan error
	client_err   error
	client_once  sync.Once
	client_codec rpc.ClientCodec
	client       *rpc.Client
}

func (control *Control) handle(
	conn_fd int,
	server *rpc.Server) {

	control_file := os.NewFile(uintptr(conn_fd), "control")
	defer control_file.Close()

	// Read single header.
	// Our header is exactly 9 characters, and we
	// expect the last character to be a newline.
	// This is a simple plaintext protocol.
	header_buf := make([]byte, 9, 9)
	n, err := control_file.Read(header_buf)
	if n != 9 || header_buf[8] != '\n' {
		if err != nil {
			control_file.Write([]byte(err.Error()))
		} else {
			control_file.Write([]byte("invalid header"))
		}
		return
	}
	header := string(header_buf)

	// We read a special header before diving into RPC
	// mode. This is because for the novmrun case, we turn
	// the socket into a stream of input/output events.
	// These are simply JSON serialized versions of the
	// events for the guest RPC interface.

	if header == "NOVM RUN\n" {

		decoder := utils.NewDecoder(control_file)
		encoder := utils.NewEncoder(control_file)

		var start noguest.StartCommand
		err := decoder.Decode(&start)
		if err != nil {
			// Poorly encoded command.
			encoder.Encode(err.Error())
			return
		}

		// Grab our client.
		client, err := control.Ready()
		if err != nil {
			encoder.Encode(err.Error())
			return
		}

		// Call start.
		result := noguest.StartResult{}
		err = client.Call("Server.Start", &start, &result)
		if err != nil {
			encoder.Encode(err.Error())
			return
		}

		// Save our pid.
		pid := result.Pid
		inputs := make(chan error)
		outputs := make(chan error)
		exitcode := make(chan int)

		// This indicates we're okay.
		encoder.Encode(nil)

		// Wait for the process to exit.
		go func() {
			wait := noguest.WaitCommand{
				Pid: pid,
			}
			var wait_result noguest.WaitResult
			err := client.Call("Server.Wait", &wait, &wait_result)
			if err != nil {
				exitcode <- 1
			} else {
				exitcode <- wait_result.Exitcode
			}
		}()

		// Read from stdout & stderr.
		go func() {
			read := noguest.ReadCommand{
				Pid: pid,
				N:   4096,
			}
			var read_result noguest.ReadResult
			for {
				err := client.Call("Server.Read", &read, &read_result)
				if err != nil {
					inputs <- err
					return
				}
				err = encoder.Encode(read_result.Data)
				if err != nil {
					inputs <- err
					return
				}
			}
		}()

		// Write to stdin.
		go func() {
			write := noguest.WriteCommand{
				Pid: pid,
			}
			var write_result noguest.WriteResult
			for {
				err := decoder.Decode(&write.Data)
				if err != nil {
					outputs <- err
					return
				}
				err = client.Call("Server.Write", &write, &write_result)
				if err != nil {
					outputs <- err
					return
				}
			}
		}()

		// Wait till exit.
		status := <-exitcode
		encoder.Encode(status)

		// Wait till EOF.
		<-inputs

		// Send a notice and close the socket.
		encoder.Encode(nil)

	} else if header == "NOVM RPC\n" {

		// Run as JSON RPC connection.
		codec := jsonrpc.NewServerCodec(control_file)
		server.ServeCodec(codec)
	}
}

func (control *Control) Serve() {

	// Bind our rpc server.
	server := rpc.NewServer()
	server.Register(control.rpc)

	for {
		// Accept clients.
		nfd, _, err := syscall.Accept(control.control_fd)
		if err == nil {
			go control.handle(nfd, server)
		}
	}
}

func NewControl(
	control_fd int,
	real_init bool,
	model *machine.Model,
	vm *platform.Vm,
	tracer *loader.Tracer,
	proxy machine.Proxy,
	is_load bool) (*Control, error) {

	// Is it invalid, for sure?
	if control_fd < 0 {
		return nil, InvalidControlSocket
	}

	// Create our control object.
	control := new(Control)
	control.control_fd = control_fd
	control.real_init = real_init
	control.proxy = proxy
	control.rpc = NewRpc(model, vm, tracer)

	// Start our barrier.
	control.client_res = make(chan error, 1)
	if is_load {
		go control.init()
	} else {
		// Already synchronized.
		control.client_res <- nil
	}

	return control, nil
}
