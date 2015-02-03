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
	"time"
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

	// Our access timestamp (for LRU).
	// This is internal and used for LRU,
	// it is not the atime or mtime on the
	// underlying file -- which is directly
	// from the underlying filesystem.
	used time.Time

	// Our index in the LRU.
	index int

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
	err := syscall.Lstat(file.write_path, &stat)
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
				err := syscall.Lstat(test_path, &stat)
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

	fs.filesLock.RLock()
	file, ok := fs.files[path]
	if ok {
		atomic.AddInt32(&file.refs, 1)
		fs.filesLock.RUnlock()
		return file, nil
	}
	fs.filesLock.RUnlock()

	// Create our new file object.
	// This isn't in the hotpath, so we
	// aren't blocking anyone else.
	newfile, err := fs.NewFile(path)

	// Escalate and create if necessary.
	fs.filesLock.Lock()
	file, ok = fs.files[path]
	if ok {
		// Race caught.
		newfile.DecRef(fs, path)
		atomic.AddInt32(&file.refs, 1)
		fs.filesLock.Unlock()
		return file, nil
	}

	if err != nil {
		fs.filesLock.Unlock()
		return nil, err
	}

	// Add the file.
	// NOTE: We add the file synchronously to the
	// LRU currently because otherwise race conditions
	// related to removing the file become very complex.
	fs.files[path] = newfile
	fs.filesLock.Unlock()

	return newfile, nil
}

func (fs *Fs) swapLru(i1 int, i2 int) {
	older_file := fs.lru[i1]

	fs.lru[i1] = fs.lru[i2]
	fs.lru[i1].index = i1

	fs.lru[i2] = older_file
	fs.lru[i2].index = i2
}

func (fs *Fs) removeLru(file *File, lock bool) {

	// This function will be called as an
	// independent goroutine in order to remove
	// a specific file (for example, on close)
	// or it will be called as a subroutine from
	// updateLru -- which is itself an synchronous
	// update function.

	if lock {
		fs.lruLock.Lock()
		defer fs.lruLock.Unlock()
	}

	// Shutdown all descriptors.
	file.flush()

	// Remove from our LRU.
	if file.index != -1 {
		if file.index == len(fs.lru)-1 {
			// Just truncate.
			fs.lru = fs.lru[0 : len(fs.lru)-1]
		} else {
			// Swap and run a bubble.
			// This may end up recursing.
			other_file := fs.lru[len(fs.lru)-1]
			fs.swapLru(file.index, len(fs.lru)-1)
			fs.lru = fs.lru[0 : len(fs.lru)-1]
			fs.updateLru(other_file, false)
		}

		// Clear our LRU index.
		file.index = -1
	}
}

func (fs *Fs) updateLru(file *File, lock bool) {

	if lock {
		fs.lruLock.Lock()
		defer fs.lruLock.Unlock()

		file.used = time.Now()

		if file.index == -1 {
			fs.lru = append(fs.lru, file)
			file.index = len(fs.lru) - 1
		}
	}

	// Not in the LRU?
	// This may be a stale update goroutine.
	if file.index == -1 {
		return
	}

	// Bubble up.
	index := file.index
	for index != 0 {
		if file.used.Before(fs.lru[index/2].used) {
			fs.swapLru(index, index/2)
			index = index / 2
			continue
		}
		break
	}

	// Bubble down.
	for index*2 < len(fs.lru) {
		if file.used.After(fs.lru[index*2].used) {
			fs.swapLru(index, index*2)
			index = index * 2
			continue
		}
		if index*2+1 < len(fs.lru) && file.used.After(fs.lru[index*2+1].used) {
			fs.swapLru(index, index*2+1)
			index = index*2 + 1
			continue
		}
		break
	}

	fs.flushLru()
}

func (fs *Fs) touchLru(file *File) {
	if file.index == -1 {
		// This needs to be done synchronously,
		// to ensure that this file is in the LRU
		// because we may have a remove() event.
		fs.updateLru(file, true)
	} else {
		// We can do this update asynchronously.
		go fs.updateLru(file, true)
	}
}

func (fs *Fs) flushLru() {
	// Are we over our limit?
	// Schedule a removal. Note that this will end
	// up recursing through updateLru() again, and
	// may end up calling flushLru() again. So we
	// don't need to check bounds, only one call.
	if len(fs.lru) > int(fs.Fdlimit) {
		fs.removeLru(fs.lru[0], false)
	}
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
	err := syscall.Lstat(file.write_path, &stat)
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
	basedir := filepath.Dir(path)

	if basedir != "." {
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
		err = file.lockWrite(fs)
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

func (file *File) rename(
	fs *Fs,
	orig_path string,
	new_path string) error {

	fs.filesLock.Lock()
	defer fs.filesLock.Unlock()

	other_file, ok := fs.files[new_path]
	if ok && other_file.exists() {
		return Eexist
	}

	// Drop the original reference.
	// (We've not replaced it atomically).
	if other_file != nil {
		defer other_file.DecRef(fs, "")
	}

	if file.write_exists && file.write_deleted {
		// Is it just marked deleted?
		err := file.unlink()
		if err != nil {
			return err
		}
	}

	// Try the rename.
	orig_read_path := file.read_path
	orig_write_path := file.write_path
	file.findPaths(fs, new_path)
	err := syscall.Rename(orig_write_path, file.write_path)
	if err != nil {
		if err == syscall.EXDEV {
			// TODO: The file cannot be renamed across file system.
			// This is a simple matter of copying the file across when
			// this happens. For now, we just return not implemented.
			err = Enotimpl
		}

		file.read_path = orig_read_path
		file.write_path = orig_write_path
		return err
	}

	// We've moved this file.
	// It didn't exist a moment ago, but it does now.
	file.write_exists = true
	file.write_deleted = false

	// Update our fids.
	// This is a bit messy, but since we are
	// holding a writeLock on this file, this
	// atomic should be reasonably atomic.
	for _, fid := range fs.Pool {
		if fid.file == file {
			fid.Path = new_path
		} else if other_file != nil && fid.file == other_file {
			// Since we hold at least one reference
			// to other_file, this should never trigger
			// a full cleanup of other_file. It's safe
			// to call DecRef here while locking the lock.
			file.IncRef(fs)
			fid.file = file
			other_file.DecRef(fs, "")
		}
	}

	// Perform the swaperoo.
	fs.files[new_path] = file
	delete(fs.files, orig_path)

	// Ensure the original file is deleted.
	// This is done at the very end, since there's
	// really nothing we can do at this point. We
	// even explicitly ignore the result. Ugh.
	setdelattr(orig_write_path)

	return nil
}

func (file *File) lockWrite(fs *Fs) error {

	file.RWMutex.RLock()
	if file.write_fd != -1 {
		fs.touchLru(file)
		return nil
	}

	// Escalate.
	file.RWMutex.RUnlock()
	file.RWMutex.Lock()
	if file.write_fd != -1 {
		// Race caught.
		file.RWMutex.Unlock()
		return file.lockWrite(fs)
	}

	// NOTE: All files are opened CLOEXEC.
	mode := syscall.O_RDWR | syscall.O_CLOEXEC
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
	return file.lockWrite(fs)
}

func (file *File) lockRead(fs *Fs) error {

	file.RWMutex.RLock()
	if file.read_fd != -1 {
		fs.touchLru(file)
		return nil
	}

	// Escalate.
	file.RWMutex.RUnlock()
	file.RWMutex.Lock()
	if file.read_fd != -1 {
		// Race caught.
		file.RWMutex.Unlock()
		return file.lockRead(fs)
	}
	if file.write_fd != -1 {
		// Use the same Fd.
		// The close logic handles this.
		file.read_fd = file.write_fd
		file.RWMutex.Unlock()
		return file.lockRead(fs)
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
	return file.lockRead(fs)
}

func (file *File) flush() {
	file.RWMutex.Lock()
	defer file.RWMutex.Unlock()

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

	file.read_fd = -1
	file.write_fd = -1
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
	err := syscall.Lstat(stat_path, &stat)
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

	// Read our symlink if available.
	if dir.Type&QTSYMLINK != 0 || dir.Mode&DMSYMLINK != 0 {
		dir.Ext, err = os.Readlink(stat_path)
		if err != nil {
			return nil, err
		}
	}

	// Plan9 doesn't handle dir+symlink.
	// We return just a raw symlink.
	if dir.Type&QTDIR != 0 && dir.Type&QTSYMLINK != 0 {
		dir.Type &= ^uint16(QTDIR)
	}
	if dir.Mode&DMDIR != 0 && dir.Mode&DMSYMLINK != 0 {
		dir.Mode &= ^uint32(DMDIR)
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
	fs.filesLock.RLock()
	atomic.AddInt32(&file.refs, 1)
	fs.filesLock.RUnlock()
}

func (file *File) DecRef(fs *Fs, path string) {

	new_refs := atomic.AddInt32(&file.refs, -1)

	if new_refs == 0 {
		fs.filesLock.Lock()
		if file.refs != 0 {
			// Race condition caught.
			fs.filesLock.Unlock()
			return
		}

		// Remove this file.
		if path != "" {
			delete(fs.files, path)
		}
		fs.filesLock.Unlock()

		// Ensure that file is removed from the LRU.
		// This will be done asynchronously, and as a
		// result all file descriptors will be closed.
		go fs.removeLru(file, true)
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

	// Clear our LRU index.
	file.index = -1

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
