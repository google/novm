package main

import (
    "flag"
    "log"
    "noguest/rpc"
    "os"
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

    // Create our RPC server.
    rpc.Run(console)
}
