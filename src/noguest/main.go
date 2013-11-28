package main

import (
    "flag"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "syscall"
    "unsafe"
)

var pivot_root = flag.String("pivot_root", "", "Pivot root filesystem.")

func pivotRoot(filesystem string) error {

    tempdir, err := ioutil.TempDir(filesystem, "old-root")
    if err != nil {
        return err
    }

    // Move into our pivot directory.
    err = os.Chdir(filesystem)
    if err != nil {
        syscall.Rmdir(tempdir)
        return err
    }

    // Take just the last part.
    _, olddir := filepath.Split(tempdir)

    // Do the actual pivot.
    _, _, e := syscall.Syscall(syscall.SYS_PIVOT_ROOT,
        uintptr(unsafe.Pointer(&([]byte(".")[0]))),
        uintptr(unsafe.Pointer(&([]byte(olddir)[0]))),
        0)
    if e != 0 {
        syscall.Rmdir(olddir)
        return e
    }

    // Unmount the old part.
    err = syscall.Unmount(olddir, 0)
    if err != nil {
        // Uh-oh, the directory is still
        // mounted, and we've pivoted.
        // Really, what can we do at this
        // point? Just pretend it's success.
        return nil
    }

    // All set.
    syscall.Rmdir(olddir)
    return nil
}

func main() {

    // Parse flags.
    flag.Parse()

    // Pivot root if necessary.
    if *pivot_root != "" {
        err := pivotRoot(*pivot_root)
        if err != nil {
            log.Fatal(err)
        }
    }
}
