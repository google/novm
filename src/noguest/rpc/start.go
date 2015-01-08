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

import (
	"bytes"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

/*
#define _XOPEN_SOURCE
#define _GNU_SOURCE
#include <stdlib.h>
#include <fcntl.h>
*/
import "C"

type StartCommand struct {

	// The command to run.
	Command []string `json:"command"`

	// The working directory.
	Cwd string `json:"cwd"`

	// Allocate a terminal?
	Terminal bool `json:"terminal"`

	// The environment.
	Environment []string `json:"environment"`
}

type StartResult struct {

	// The resulting pid.
	Pid int `json:"pid"`
}

func (server *Server) Start(
	command *StartCommand,
	result *StartResult) error {

	// We need at least a command.
	if len(command.Command) == 0 {
		return syscall.EINVAL
	}

	// Lookup our binary name.
	var binary string
	_, err := os.Stat(command.Command[0])
	if err == nil {
		// Absolute path is okay.
		binary = command.Command[0]
	} else {
		// Check our environment.
		for _, keyval := range command.Environment {
			if strings.HasPrefix(keyval, "PATH=") && len(keyval) > 5 {
				dirpaths := strings.Split(keyval[5:], ":")
				for _, dirpath := range dirpaths {
					testpath := path.Join(dirpath, command.Command[0])
					_, err = os.Stat(testpath)
					if err == nil {
						binary = testpath
						break
					}
				}
			}
		}
	}

	// Did we find a binary?
	if binary == "" {
		return syscall.ENOENT
	}

	var input *os.File
	var output *os.File

	var stdin *os.File
	var stdout *os.File
	var stderr *os.File

	if command.Terminal {
		// Open a master terminal Fd.
		fd, err := C.posix_openpt(syscall.O_RDWR | syscall.O_NOCTTY)
		if fd == C.int(-1) && err != nil {
			// Out of FDs?
			result.Pid = -1
			return err
		}

		// Save our master.
		master := os.NewFile(uintptr(fd), "ptmx")

		// Try to grant and unlock the PT.
		r, err := C.grantpt(C.int(master.Fd()))
		if r != C.int(0) && err != nil {
			master.Close()
			result.Pid = -1
			return err
		}
		r, err = C.unlockpt(C.int(master.Fd()))
		if r != C.int(0) && err != nil {
			master.Close()
			result.Pid = -1
			return err
		}

		// Get the terminal name.
		buf := make([]byte, 1024, 1024)
		r, err = C.ptsname_r(
			C.int(master.Fd()),
			(*C.char)(unsafe.Pointer(&buf[0])),
			1024)
		if r != C.int(0) && err != nil {
			master.Close()
			result.Pid = -1
			return err
		}

		// Open the slave terminal.
		n := bytes.Index(buf, []byte{0})
		slave_pts := string(buf[:n])
		slave, err := os.OpenFile(slave_pts, syscall.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			master.Close()
			result.Pid = -1
			return err
		}

		defer slave.Close()

		// Set our inputs.
		input = master
		output = master
		stdin = slave
		stdout = slave
		stderr = slave

	} else {
		// Allocate pipes.
		r1, w1, err := os.Pipe()
		if err != nil {
			result.Pid = -1
			return err
		}
		r2, w2, err := os.Pipe()
		if err != nil {
			r1.Close()
			w1.Close()
			result.Pid = -1
			return err
		}

		defer r1.Close()
		defer w2.Close()

		// Set our inputs.
		input = w1
		output = r2
		stdin = r1
		stdout = w2
		stderr = w2
	}

	// Start the process.
	proc_attr := &os.ProcAttr{
		Dir:   command.Cwd,
		Env:   command.Environment,
		Files: []*os.File{stdin, stdout, stderr},
		Sys: &syscall.SysProcAttr{
			Setsid:  true,
			Setctty: command.Terminal,
			Ctty:    0,
		},
	}
	proc, err := os.StartProcess(
		binary,
		command.Command,
		proc_attr)

	// Unable to start?
	if err != nil {
		input.Close()
		if input != output {
			output.Close()
		}
		result.Pid = -1
		return err
	}

	// Create our process.
	process := &Process{
		input:     input,
		output:    output,
		starttime: time.Now(),
		cond:      sync.NewCond(&sync.Mutex{}),
	}

	// Save the pid.
	result.Pid = proc.Pid

	server.mutex.Lock()
	old_process := server.active[result.Pid]
	server.active[result.Pid] = process
	server.mutex.Unlock()

	if old_process != nil {
		old_process.close()
	}

	go server.wait()
	return nil
}
