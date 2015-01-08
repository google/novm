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
	"math"
	"novmm/platform"
	"path"
	"syscall"
)

func (fs *Fs) version(
	msize uint32,
	version string) (uint32, bool, string, error) {

	// Cap the msize.
	if msize < IOHDRSZ {
		return 0, false, "", &Error{"msize too small", EINVAL}
	}

	// We speak basic P9 or P9.u.
	dotu := version == "9P2000.u" && fs.Dotu
	ver := "9P2000"
	if dotu {
		ver = "9P2000.u"
	}

	return msize, dotu, ver, nil
}

func (fs *Fs) versionPost(
	msize uint32,
	dotu bool) {

	// Save whether we are using dotu.
	fs.Dotu = dotu
}

func (fs *Fs) auth(
	afid uint32,
	uname string,
	aname string,
	unamenum uint32) (*Qid, error) {

	// We don't support any authentication.
	return nil, Enotimpl
}

func (fs *Fs) attach(
	fid uint32,
	afid *Fid,
	uname string,
	aname string,
	unamenum uint32) (*Qid, error) {

	// As per above,
	// We ignore all auth parameters.
	// -> No auth fid,
	// -> No uname
	// -> No aname
	// -> No unamenum

	// Create the fid.
	fs.root.IncRef(fs)
	newfid, err := fs.NewFid(fid, "/", fs.root)
	if err != nil {
		if newfid != nil {
			newfid.DecRef(fs)
		} else {
			fs.root.DecRef(fs, "/")
		}
		return nil, err
	}

	return &fs.root.Qid, nil
}

func (fs *Fs) walk(
	fid *Fid,
	newfid uint32,
	names []string) ([]Qid, error) {

	// Do the actual walk.
	qids := make([]Qid, 0, len(names))
	fullpath := fid.Path
	file := fid.file
	file.IncRef(fs)

	for _, name := range names {
		// Calculate the resulting path.
		// NOTE: The Join() operation will also
		// "Clean" the path i.e. canonicalize.
		file.DecRef(fs, fullpath)
		fullpath = path.Join(fullpath, name)

		var err error
		file, err = fs.lookup(fullpath)
		if err != nil {
			if file != nil {
				file.DecRef(fs, fullpath)
			}
			return nil, err
		}

		// Can this file not exist?
		if !file.exists() {
			file.DecRef(fs, fullpath)
			return nil, Enoent
		}

		qids = append(qids, file.Qid)
	}

	// Is this changing the original fid?
	var nfid *Fid
	var err error
	if newfid == fid.Fid {
		// Drop the original reference,
		// since we have an extra from above.
		fid.file.DecRef(fs, fid.Path)

		// Reset the state of the file.
		fid.Path = fullpath
		fid.file = file
		fid.Opened = false
		fid.Omode = 0
		nfid = fid

	} else {
		nfid, err = fs.NewFid(newfid, fullpath, file)
		if err != nil {
			if nfid != nil {
				nfid.DecRef(fs)
			} else {
				file.DecRef(fs, fullpath)
			}
			return nil, err
		}
	}

	return qids, nil
}

func (fs *Fs) open(
	fid *Fid,
	mode uint8) (*Qid, uint32, error) {

	// Already opened?
	if fid.Opened {
		return nil, 0, Eopen
	}

	// Trying to open a directory for writing?
	if (fid.file.Type&QTDIR) != 0 && mode != OREAD {
		return nil, 0, Eperm
	}

	return &fid.file.Qid, platform.PageSize, nil
}

func (fs *Fs) openPost(
	fid *Fid,
	mode uint8) {

	fid.Omode = mode
	fid.Opened = true
}

func (fs *Fs) create(
	fid *Fid,
	name string,
	perm uint32,
	mode uint8,
	ext string) (*File, *Qid, uint32, error) {

	// Already opened?
	if fid.Opened {
		return nil, nil, 0, Eopen
	}

	// Trying to create not in a directory?
	if (fid.file.Qid.Type & QTDIR) == 0 {
		return nil, nil, 0, Enotdir
	}

	// FIXME: We currently ignore permissions on
	// this directory. We should be checking that
	// this user can write to this directory.

	// Can't create special files if not 9P2000.u.
	if perm&(DMNAMEDPIPE|DMSYMLINK|DMLINK|DMDEVICE|DMSOCKET) != 0 && !fs.Dotu {
		return nil, nil, 0, Eperm
	}

	// FIXME: We don't currently support creation of
	// symlinks. This should be relatively straight-forward,
	// but needs to be appropriately plumbed through. This
	// work will be done in a future commit.
	if perm&(DMNAMEDPIPE|DMSYMLINK|DMLINK|DMDEVICE|DMSOCKET) != 0 {
		return nil, nil, 0, Enotimpl
	}

	// Compute our new mode.
	file_mode := uint32(0)
	for mask, mode_bit := range P9ModeToMode {
		if mask&perm == mask {
			file_mode |= mode_bit
		}
	}
	for mask, mode_bit := range P9TypeToMode {
		if mask&mode == mask {
			file_mode |= mode_bit
		}
	}

	// Let's actually create the file.
	new_path := path.Join(fid.Path, name)
	new_file, err := fs.lookup(new_path)
	if err != nil {
		if new_file != nil {
			new_file.DecRef(fs, path.Join(fid.Path, name))
		}
		return nil, nil, 0, err
	}
	err = new_file.create(fs, new_path, file_mode)
	if err != nil {
		new_file.DecRef(fs, path.Join(fid.Path, name))
		return nil, nil, 0, err
	}

	// Give back a reference to our file.
	return new_file, &new_file.Qid, platform.PageSize, err
}

func (fs *Fs) createPost(
	fid *Fid,
	new_file *File,
	name string,
	mode uint8) {

	// Swap out the files.
	fid.file.DecRef(fs, fid.Path)
	fid.file = new_file

	fid.Omode = mode
	fid.Opened = true
}

func (fs *Fs) createFail(
	fid *Fid,
	file *File,
	name string,
	mode uint8) {

	if file != nil {
		// Drop the new file.
		file.DecRef(fs, path.Join(fid.Path, name))
	}
}

func (fs *Fs) readDir(
	fid *Fid,
	offset int64,
	length int) ([]*Dir, error) {

	// Assume this is a directory.
	// NOTE: This is a bit of a ugly
	// special case in the main fs loop.

	if offset != 0 && offset != int64(fid.Diroffset) {
		return nil, Ebadoffset
	}
	if offset == 0 {
		fid.Diroffset = 0

		// We generate a list of children.
		// This is cached for future reads.
		var err error
		fid.Direntries, err = fid.file.children(fs, fid.Path)
		if err != nil {
			return nil, err
		}
	}

	// Exhausted?
	if fid.Direntries == nil {
		return []*Dir{}, nil
	}

	return fid.Direntries, nil
}

func (fs *Fs) readFile(
	fid *Fid,
	offset int64,
	length int) (int, error) {

	// Assume this is a regular file.
	// NOTE: This is a bit of a ugly
	// special case in the main fs loop.

	// Lock the file for reading.
	err := fid.file.lockRead(fs)
	return fid.file.read_fd, err
}

func (fs *Fs) readDirPost(
	fid *Fid,
	count uint32,
	entries int) {

	if fid.file.Qid.Type&QTDIR != 0 {
		fid.Diroffset += uint64(count)
		if fid.Direntries != nil &&
			entries < len(fid.Direntries) {
			fid.Direntries = fid.Direntries[entries:]
		} else {
			fid.Direntries = nil
		}
	}
}

func (fs *Fs) readFilePost(
	fid *Fid,
	count uint32) {

	fid.file.unlock()
}

func (fs *Fs) readFileFail(
	fid *Fid,
	count uint32) {

	fid.file.unlock()
}

func (fs *Fs) writeFile(
	fid *Fid,
	offset int64,
	count int) (int, error) {

	err := fid.file.lockWrite(fs)
	return fid.file.write_fd, err
}

func (fs *Fs) writeFilePost(
	fid *Fid,
	count uint32) {

	fid.file.unlock()
}

func (fs *Fs) writeFileFail(
	fid *Fid,
	count uint32) {

	fid.file.unlock()
}

func (fs *Fs) clunk(fid *Fid) error {
	return nil
}

func (fs *Fs) clunkPost(fid *Fid) {

	// Drop the fid reference.
	// This should remove it from the map,
	// (once all related requests are done).
	fid.DecRef(fs)
}

func (fs *Fs) remove(fid *Fid) error {

	// It seems that we universally drop
	// a reference to the Fid here. The handler
	// maintains a single reference to the Fid,
	// so this will cause it disappear regardless
	// of whether an error occurs or not. This
	// appears to be the correct behaviour.
	fid.DecRef(fs)

	return nil
}

func (fs *Fs) removePost(fid *Fid) error {

	// Unlink if the file is there.
	return fid.file.remove(fs, fid.Path)
}

func (fs *Fs) stat(fid *Fid) (*Dir, error) {

	// Get underlying file information.
	return fid.file.dir(path.Base(fid.Path), true)
}

func (fs *Fs) wstat(fid *Fid, next *Dir) (*Dir, error) {

	// Can we write to the file?
	err := fid.file.lockWrite(fs)
	if err != nil {
		return nil, err
	}

	// Grab underlying information.
	cur, err := fs.stat(fid)
	if err != nil {
		fid.file.unlock()
		return nil, err
	}

	if next.Muidnum != math.MaxUint32 &&
		next.Muidnum != cur.Muidnum {
		// Not supported.
		fid.file.unlock()
		return nil, Enotimpl
	}

	return cur, nil
}

func (fs *Fs) wstatPost(fid *Fid, cur *Dir, next *Dir) error {

	defer fid.file.unlock()

	// Rename the file.
	if next.Name != "" &&
		next.Name != cur.Name {

		new_path := path.Join(path.Dir(fid.Path), next.Name)
		err := fid.file.rename(fs, fid.Path, new_path)
		if err != nil {
			return err
		}
	}

	// Update our access times.
	atime := cur.Atime
	mtime := cur.Atime
	if next.Atime != math.MaxUint32 {
		atime = next.Atime
	}
	if next.Mtime != math.MaxUint32 {
		mtime = next.Mtime
	}
	if atime != cur.Atime || mtime != cur.Mtime {
		err := syscall.Futimes(
			fid.file.write_fd,
			[]syscall.Timeval{
				syscall.Timeval{int64(atime), 0},
				syscall.Timeval{int64(mtime), 0},
			})
		if err != nil {
			return err
		}
	}

	// Truncate the file.
	if next.Length != math.MaxUint64 &&
		next.Length != cur.Length {
		err := syscall.Ftruncate(fid.file.write_fd, int64(next.Length))
		if err != nil {
			return err
		}
	}

	// Change the owner.
	uid := cur.Uidnum
	gid := cur.Gidnum
	if next.Uidnum != math.MaxUint32 &&
		next.Uidnum != cur.Uidnum {
		uid = next.Uidnum
	}
	if next.Gidnum != math.MaxUint32 &&
		next.Gidnum != cur.Gidnum {
		gid = next.Gidnum
	}
	if uid != cur.Uidnum || gid != cur.Gidnum {
		err := syscall.Fchown(fid.file.write_fd, int(uid), int(gid))
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *Fs) wstatFail(fid *Fid, cur *Dir, next *Dir) {
	fid.file.unlock()
}
