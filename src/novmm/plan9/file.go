// Copyright 2009 The Go9p Authors.  All rights reserved.
// Copyright 2013 Adin Scannell.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the licenses/go9p file.
package plan9

import (
    "io/ioutil"
    "os"
    "path"
    "path/filepath"
    "strings"
    "sync"
    "sync/atomic"
    "syscall"
)

type File struct {

    // File identifier.
    Qid

    // The number of fid references.
    refs int32

    // Our real underlying read/write paths.
    read_path     string
    read_exists   bool
    write_path    string
    write_exists  bool
    write_deleted bool

    // Our underlying mode.
    mode uint32

    // The associated file fds.
    read_fd  int
    write_fd int

    // The write map --
    //
    // This is future work.
    //
    // This will track sparse holes in the write_fd,
    // and ensures that it is populated as necessary.
    // Each entry represents a sparse hole. If a read
    // comes in and corresponds to a hole, we will send
    // the read to the read_fd. If a read comes in and
    // partially overlaps with a hole, then we need to
    // copy data from the read_fd to the write_fd first,
    // then return the write_fd. When a write comes in,
    // we always send the write to the write_fd and
    // update the write_map appropriately to remove any
    // holes that might be there.
    //
    // NOTE: The write files are actually *sparse*
    // copies on top of the read files. It's very
    // important that tar -S is used to compress and
    // uncompress bundles to have this maintained.
    //
    // type Hole struct {
    //  start  uint64
    //  length uint64
    // }
    //
    // write_map []Hole

    // Our RWMutex (protects r=>w transition).
    sync.RWMutex
}

var ModeToP9Type = map[uint32]uint16{
    syscall.S_IFDIR: QTDIR,
    syscall.S_IFLNK: QTSYMLINK,
}

var P9TypeToMode = map[uint8]uint32{
    QTDIR:     syscall.S_IFDIR,
    QTSYMLINK: syscall.S_IFLNK,
}

var ModeToP9Mode = map[uint32]uint32{
    syscall.S_IFDIR:  DMDIR,
    syscall.S_IFLNK:  DMSYMLINK,
    syscall.S_IFSOCK: DMSOCKET,
    syscall.S_IFBLK:  DMDEVICE,
    syscall.S_IFCHR:  DMDEVICE,
    syscall.S_ISUID:  DMSETUID,
    syscall.S_ISGID:  DMSETGID,
    syscall.S_IRUSR:  DMREAD << 6,
    syscall.S_IWUSR:  DMWRITE << 6,
    syscall.S_IXUSR:  DMEXEC << 6,
    syscall.S_IRGRP:  DMREAD << 3,
    syscall.S_IWGRP:  DMWRITE << 3,
    syscall.S_IXGRP:  DMEXEC << 3,
    syscall.S_IROTH:  DMREAD,
    syscall.S_IWOTH:  DMWRITE,
    syscall.S_IXOTH:  DMEXEC,
}

var P9ModeToMode = map[uint32]uint32{
    DMDIR:          syscall.S_IFDIR,
    DMSYMLINK:      syscall.S_IFLNK,
    DMSOCKET:       syscall.S_IFSOCK,
    DMDEVICE:       syscall.S_IFCHR,
    DMSETUID:       syscall.S_ISUID,
    DMSETGID:       syscall.S_ISGID,
    (DMREAD << 6):  syscall.S_IRUSR,
    (DMWRITE << 6): syscall.S_IWUSR,
    (DMEXEC << 6):  syscall.S_IXUSR,
    (DMREAD << 3):  syscall.S_IRGRP,
    (DMWRITE << 3): syscall.S_IWGRP,
    (DMEXEC << 3):  syscall.S_IXGRP,
    DMREAD:         syscall.S_IROTH,
    DMWRITE:        syscall.S_IWOTH,
    DMEXEC:         syscall.S_IXOTH,
}

func (file *File) findPaths(fs *Fs, filepath string) {

    // Figure out our write path first.
    write_prefix := ""
    write_backing_path := "."

    for prefix, backing_path := range fs.Write {
        if strings.HasPrefix(filepath, prefix) &&
            len(prefix) > len(write_prefix) {
            write_prefix = prefix
            write_backing_path = backing_path
        }
    }
    file.write_path = path.Join(
        write_backing_path,
        filepath[len(write_prefix):])

    var stat syscall.Stat_t
    err := syscall.Stat(file.write_path, &stat)
    if err == nil {
        file.write_exists = true
        file.write_deleted, _ = readdelattr(file.write_path)
        if !file.write_deleted {
            file.mode = stat.Mode
        }
    } else {
        file.write_exists = false
        file.write_deleted = false
    }

    // Figure out our read path.
    read_prefix := write_prefix
    read_backing_path := write_backing_path
    file.read_exists = false

    for prefix, backing_paths := range fs.Read {
        if strings.HasPrefix(filepath, prefix) &&
            (!file.read_exists ||
                len(prefix) > len(read_prefix)) {

            for _, backing_path := range backing_paths {

                // Does this file exist?
                test_path := path.Join(backing_path, filepath[len(prefix):])
                err := syscall.Stat(test_path, &stat)
                if err == nil {
                    // Check if it's deleted.
                    // NOTE: If we can't read the extended
                    // attributes on this file, we can assume
                    // that it is not deleted.
                    deleted, _ := readdelattr(test_path)
                    if !deleted {
                        read_prefix = prefix
                        read_backing_path = backing_path
                        file.read_exists = true
                        if !file.write_deleted && !file.write_exists {
                            file.mode = stat.Mode
                        }
                    }
                }
            }
        }
    }
    file.read_path = path.Join(
        read_backing_path,
        filepath[len(read_prefix):])
}

func (fs *Fs) lookup(path string) (*File, error) {

    // Normalize path.
    if len(path) > 0 && path[len(path)-1] == '/' {
        path = path[:len(path)-1]
    }

    fs.fileLock.RLock()
    file, ok := fs.files[path]
    if ok {
        atomic.AddInt32(&file.refs, 1)
        fs.fileLock.RUnlock()
        return file, nil
    }
    fs.fileLock.RUnlock()

    // Create our new file object.
    // This isn't in the hotpath, so we
    // aren't blocking anyone else.
    newfile, err := fs.NewFile(path)

    // Escalate and create if necessary.
    fs.fileLock.Lock()
    file, ok = fs.files[path]
    if ok {
        // Race caught.
        newfile.DecRef(fs, path)
        atomic.AddInt32(&file.refs, 1)
        fs.fileLock.Unlock()
        return file, nil
    }

    if err != nil {
        fs.fileLock.Unlock()
        return nil, err
    }

    fs.files[path] = newfile
    fs.fileLock.Unlock()

    return newfile, nil
}

func (file *File) unlink() error {

    // Remove whatever was there.
    // NOTE: We will generally require the
    // write lock to be held for this routine.

    if file.write_deleted {
        err := cleardelattr(file.write_path)
        if err != nil {
            return err
        }
    }

    var stat syscall.Stat_t
    err := syscall.Stat(file.write_path, &stat)
    if err == nil {
        if stat.Mode&syscall.S_IFDIR != 0 {
            err = syscall.Rmdir(file.write_path)
            if err != nil {
                return err
            }
        } else {
            err = syscall.Unlink(file.write_path)
            if err != nil {
                return err
            }
        }
    }

    file.write_exists = false
    file.write_deleted = false

    return nil
}

func (file *File) remove(
    fs *Fs,
    path string) error {

    file.RWMutex.Lock()
    defer file.RWMutex.Unlock()

    // Unlink what's there.
    err := file.unlink()
    if err != nil {
        return err
    }

    // Make sure the parent exists.
    err = file.makeTree(fs, path)
    if err != nil {
        file.RWMutex.Unlock()
        return err
    }

    // We need to have something we can record
    // on. Even for files we record a directory,
    // this later on packs may choose to make this
    // into a tree and we need to be ready for that.
    mode := (syscall.S_IFDIR | syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR)
    err = syscall.Mkdir(file.write_path, uint32(mode))
    if err != nil {
        return err
    }

    // Mark this file as deleted.
    err = setdelattr(file.write_path)
    if err != nil {
        return err
    }

    // We're deleted.
    file.write_exists = true
    file.write_deleted = true

    return nil
}

func (file *File) exists() bool {
    // Some file must exist.
    return (!file.write_deleted &&
        (file.read_exists || file.write_exists))
}

func (file *File) makeTree(
    fs *Fs,
    path string) error {

    // Make all the super directories.
    basedir, _ := filepath.Split(path)

    if basedir != path {
        parent, err := fs.lookup(basedir)
        if err != nil {
            return err
        }

        // The parent must have had
        // a valid mode set at some point.
        // We ignore this error, as this
        // may actually return Eexist.
        parent.create(fs, basedir, parent.mode)
        parent.DecRef(fs, basedir)
    }

    return nil
}

func (file *File) create(
    fs *Fs,
    path string,
    mode uint32) error {

    file.RWMutex.Lock()
    did_exist := file.exists()
    if file.write_exists && !file.write_deleted {
        file.RWMutex.Unlock()
        return Eexist
    }

    // Save our mode.
    file.mode = mode

    // Is it a directory?
    if file.mode&syscall.S_IFDIR != 0 {

        if file.write_exists && file.write_deleted {
            // Is it just marked deleted?
            err := file.unlink()
            if err != nil {
                file.RWMutex.Unlock()
                return err
            }
        }

        // Make sure the parent exists.
        err := file.makeTree(fs, path)
        if err != nil {
            file.RWMutex.Unlock()
            return err
        }

        // Make this directory.
        err = syscall.Mkdir(file.write_path, mode)
        if err != nil {
            file.RWMutex.Unlock()
            return err
        }

        // Fill out type.
        err = file.fillType(file.write_path)
        if err != nil {
            file.RWMutex.Unlock()
            return err
        }

        // We now exist.
        file.write_exists = true
        file.RWMutex.Unlock()

    } else {
        // Make sure the parent exists.
        err := file.makeTree(fs, path)
        if err != nil {
            file.RWMutex.Unlock()
            return err
        }

        file.RWMutex.Unlock()
        err = file.lockWrite(0, 0)
        if err != nil {
            return err
        }

        err = file.fillType(file.write_path)
        if err != nil {
            file.unlock()
            return err
        }

        file.unlock()
    }

    if did_exist {
        return Eexist
    }

    return nil
}

func (file *File) lockWrite(
    offset int64,
    length int) error {

    file.RWMutex.RLock()
    if file.write_fd != -1 {
        return nil
    }

    // Escalate.
    file.RWMutex.RUnlock()
    file.RWMutex.Lock()
    if file.write_fd != -1 {
        // Race caught.
        file.RWMutex.Unlock()
        return file.lockWrite(offset, length)
    }

    mode := syscall.O_RDWR
    var perm uint32

    // Make sure the file exists.
    if !file.write_exists || file.write_deleted {
        // NOTE: It would be really great to handle
        // all these writes as simply overlays and keep
        // a map of all the sparse holes in the file.
        // See above with the write_map, for now I'll
        // leave this for future work.
        if file.write_deleted {
            // Remove the file.
            file.unlink()
            mode |= syscall.O_RDWR | syscall.O_CREAT
            perm |= syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
            file.write_deleted = false
            file.write_exists = true

        } else if !file.read_exists {
            // This is a fresh file.
            // It doesn't exist in any read layer.
            mode |= syscall.O_CREAT | syscall.O_RDWR
            perm |= syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
            file.write_exists = true

        } else {
            // Not deleted && read_exists.
            // We grab a memory map and write out
            // a copy of the new file. This could
            // be made much more efficient (per above).
            data, err := ioutil.ReadFile(file.read_path)
            if err != nil {
                file.RWMutex.Unlock()
                return err
            }
            perm |= syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IXUSR
            err = ioutil.WriteFile(file.write_path, data, os.FileMode(perm))
            if err != nil {
                file.RWMutex.Unlock()
                return err
            }
            file.write_exists = true
        }
    }

    new_fd, err := syscall.Open(file.write_path, mode, perm)
    if err != nil {
        file.RWMutex.Unlock()
        return err
    }

    // Open successful.
    file.write_fd = new_fd

    // Flush the current readFD.
    if file.read_fd != -1 {
        syscall.Close(file.read_fd)
        file.read_fd = -1
    }

    // Retry (for the RLock).
    file.RWMutex.Unlock()
    return file.lockWrite(offset, length)
}

func (file *File) lockRead(
    offset int64,
    length int) error {

    file.RWMutex.RLock()
    if file.read_fd != -1 {
        return nil
    }

    // Escalate.
    file.RWMutex.RUnlock()
    file.RWMutex.Lock()
    if file.read_fd != -1 {
        // Race caught.
        file.RWMutex.Unlock()
        return file.lockRead(offset, length)
    }
    if file.write_fd != -1 {
        // Use the same Fd.
        // The close logic handles this.
        file.read_fd = file.write_fd
        file.RWMutex.Unlock()
        return file.lockRead(offset, length)
    }

    // Okay, no write available.
    // Let's open our read path.
    new_fd, err := syscall.Open(file.read_path, syscall.O_RDONLY, 0)
    if err != nil {
        file.RWMutex.Unlock()
        return err
    }

    // Open successful.
    file.read_fd = new_fd

    // Retry (for the RLock).
    file.RWMutex.Unlock()
    return file.lockRead(offset, length)
}

func (file *File) dir(
    name string,
    locked bool) (*Dir, error) {

    if locked {
        file.RWMutex.RLock()
    }
    var stat_path string
    if file.write_exists {
        stat_path = file.write_path
    } else {
        stat_path = file.read_path
    }

    var stat syscall.Stat_t
    err := syscall.Stat(stat_path, &stat)
    if locked {
        file.RWMutex.RUnlock()
    }
    if err != nil {
        return nil, err
    }

    dir := new(Dir)
    dir.Type = 0 // Set below.
    dir.Mode = 0 // Set below.
    dir.Qid = file.Qid
    dir.Dev = uint32(stat.Dev)
    atim, _ := stat.Atim.Unix()
    dir.Atime = uint32(atim)
    mtim, _ := stat.Mtim.Unix()
    dir.Mtime = uint32(mtim)
    if stat.Mode&syscall.S_IFDIR != 0 {
        dir.Length = 0
    } else {
        dir.Length = uint64(stat.Size)
    }
    dir.Name = name
    dir.Uid = "root"
    dir.Gid = "root"
    dir.Muid = "root"
    dir.Ext = ""
    dir.Uidnum = stat.Uid
    dir.Gidnum = stat.Gid
    dir.Muidnum = stat.Uid

    for mask, type_bit := range ModeToP9Type {
        if stat.Mode&mask == mask {
            dir.Type = dir.Type | type_bit
        }
    }
    for mask, mode_bit := range ModeToP9Mode {
        if stat.Mode&mask == mask {
            dir.Mode = dir.Mode | mode_bit
        }
    }

    return dir, nil
}

func (file *File) children(fs *Fs, dirpath string) ([]*Dir, error) {

    child_set := make(map[string]bool)
    gather_dir := func(realdir string) {
        files, err := filepath.Glob(path.Join(realdir, "*"))
        if err != nil {
            return
        }
        for _, file := range files {
            // This file exists somewhere.
            child_set[path.Base(file)] = true
        }
    }

    // We need to collect all possible matching paths.
    // This has the potential to be a very long list.
    for prefix, backing_path := range fs.Write {
        if strings.HasPrefix(dirpath, prefix) {
            gather_dir(path.Join(backing_path, dirpath[len(prefix):]))
        }
    }
    for prefix, backing_paths := range fs.Read {
        if strings.HasPrefix(dirpath, prefix) {
            for _, backing_path := range backing_paths {
                gather_dir(path.Join(backing_path, dirpath[len(prefix):]))
            }
        }
    }

    // We stat each of these files.
    results := make([]*Dir, 0, len(child_set))
    for name, _ := range child_set {

        // Find this child.
        child_path := path.Join(dirpath, name)
        child, err := fs.lookup(child_path)
        if err != nil {
            if child != nil {
                child.DecRef(fs, child_path)
            }
            return nil, err
        }

        // Deleted?
        if !child.exists() {
            child.DecRef(fs, child_path)
            continue
        }

        // Get the stat.
        child_dir, err := child.dir(name, true)
        child.DecRef(fs, child_path)
        if err != nil {
            return nil, err
        }

        results = append(results, child_dir)
    }

    // We're good.
    return results, nil
}

func (file *File) unlock() {
    file.RWMutex.RUnlock()
}

func (file *File) IncRef(fs *Fs) {
    fs.fileLock.RLock()
    atomic.AddInt32(&file.refs, 1)
    fs.fileLock.RUnlock()
}

func (file *File) DecRef(fs *Fs, path string) {

    new_refs := atomic.AddInt32(&file.refs, -1)

    if new_refs == 0 {
        fs.fileLock.Lock()
        if file.refs != 0 {
            // Race condition caught.
            fs.fileLock.Unlock()
            return
        }

        // Remove this file.
        delete(fs.files, path)
        fs.fileLock.Unlock()

        // Close the file if still opened.
        if file.read_fd != -1 {
            syscall.Close(file.read_fd)
        }

        // Close the write_fd if it's open
        // (and it's unique).
        if file.write_fd != -1 &&
            file.write_fd != file.read_fd {
            syscall.Close(file.write_fd)
        }
    }
}

func (file *File) fillType(path string) error {

    // Figure out the type.
    dir, err := file.dir(path, false)
    if err != nil {
        return err
    }

    // Get file type.
    file.Qid.Type = uint8(dir.Type)
    return nil
}

func (fs *Fs) NewFile(path string) (*File, error) {

    file := new(File)
    file.refs = 1

    // Figure out the paths.
    file.findPaths(fs, path)

    // Reset our FDs.
    file.read_fd = -1
    file.write_fd = -1

    file.Qid.Version = 0
    file.Qid.Path = atomic.AddUint64(&fs.Fileid, 1)

    if file.exists() {
        return file, file.fillType(path)
    }

    return file, nil
}
