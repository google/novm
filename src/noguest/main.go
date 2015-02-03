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

package main

import (
	"flag"
	"log"
	"noguest/protocol"
	"noguest/rpc"
	"os"
	"os/exec"
	"syscall"
)

// The default control file.
var control = flag.String("control", "/dev/vport0p0", "control file")

// Should this always run a server.
var server_fd = flag.Int("serverfd", -1, "run RPC server")

func mount(fs string, location string) error {

	// Do we have the location?
	_, err := os.Stat(location)
	if err != nil {
		// Make sure it's a directory.
		err = os.Mkdir(location, 0755)
		if err != nil {
			return err
		}
	}

	// Try to mount it.
	cmd := exec.Command("/bin/mount", "-t", fs, fs, location)
	return cmd.Run()
}

func main() {
	var console *os.File

	// Parse flags.
	flag.Parse()

	if *server_fd == -1 {
		// Open the console.
		if f, err := os.OpenFile(*control, os.O_RDWR, 0); err != nil {
			log.Fatal("Problem opening console:", err)
		} else {
			console = f
		}

		// Make sure devpts is mounted.
		err := mount("devpts", "/dev/pts")
		if err != nil {
			log.Fatal(err)
		}

		// Notify novmm that we're ready.
		buffer := make([]byte, 1, 1)
		buffer[0] = protocol.NoGuestStatusOkay
		n, err := console.Write(buffer)
		if err != nil || n != 1 {
			log.Fatal(err)
		}

		// Read our response.
		n, err = console.Read(buffer)
		if n != 1 || err != nil {
			log.Fatal(protocol.UnknownCommand)
		}

		// Rerun to cleanup argv[0], or create a real init.
		new_args := make([]string, 0, len(os.Args)+1)
		new_args = append(new_args, "noguest")
		new_args = append(new_args, "-serverfd", "0")
		new_args = append(new_args, os.Args[1:]...)

		switch buffer[0] {

		case protocol.NoGuestCommandRealInit:
			// Run our noguest server in a new process.
			proc_attr := &syscall.ProcAttr{
				Dir:   "/",
				Env:   os.Environ(),
				Files: []uintptr{console.Fd(), 1, 2},
			}
			_, err := syscall.ForkExec(os.Args[0], new_args, proc_attr)
			if err != nil {
				log.Fatal(err)
			}

			// Exec our real init here in place.
			err = syscall.Exec("/sbin/init", []string{"init"}, os.Environ())
			log.Fatal(err)

		case protocol.NoGuestCommandFakeInit:
			// Since we don't have any init to setup basic
			// things, like our hostname we do some of that here.
			syscall.Sethostname([]byte("novm"))

		default:
			// What the heck is this?
			log.Fatal(protocol.UnknownCommand)
		}
	} else {
		// Open the defined fd.
		console = os.NewFile(uintptr(*server_fd), "console")
	}

	// Small victory.
	log.Printf("~~~ NOGUEST ~~~")

	// Create our RPC server.
	rpc.Run(console)
}
