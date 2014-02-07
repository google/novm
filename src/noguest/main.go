package main

import (
    "flag"
    "log"
    "noguest/rpc"
    "os"
    "os/exec"
)

var control = flag.String("control", "/dev/vport0p0", "control file")

func main() {

    // Parse flags.
    flag.Parse()

    // Small victory.
    log.Printf("~~~ NOGUEST ~~~")

    // Open the console.
    console, err := os.OpenFile(*control, os.O_RDWR, 0)
    if err != nil {
        log.Fatal("Problem opening console:", err)
    }

    // Do we have a /dev/pts?
    _, err = os.Stat("/dev/pts")
    if err != nil {
        // Make sure it's a directory.
        err = os.Mkdir("/dev/pts", 0755)
        if err != nil {
            log.Fatal(err)
        }

        // Make sure /dev/pts is mounted.
        cmd := exec.Command("/bin/mount", "-t", "devpts", "devpts", "/dev/pts")
        err = cmd.Run()
        if err != nil {
            log.Fatal(err)
        }
    }

    // Notify novmm that we're ready.
    // This is a very simple barrier, but
    // should be good enough for now.
    n, err := console.Write([]byte{0x42})
    if err != nil || n != 1 {
        log.Fatal(err)
    }

    // Create our RPC server.
    rpc.Run(console)
}
