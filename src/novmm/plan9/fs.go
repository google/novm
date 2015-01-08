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
	"encoding/json"
	"log"
	"sync"
	"syscall"
)

//
// Outstanding request list.
//
type Reqlist map[uint16]bool

//
// Filesystem state.
//
type Fs struct {

	// The read mappings.
	Read map[string][]string `json:"read"`

	// The write mappings.
	Write map[string]string `json:"write"`

	// are we speaking 9P2000.u?
	Dotu bool `json:"dotu"`

	// open FiDs.
	Pool Fidpool `json:"pool"`

	// requests outstanding.
	Reqs Reqlist `json"reqs"`

	// Lock protecting the above.
	fidLock sync.RWMutex
	fidCond *sync.Cond

	// Our open files.
	// This is not serialized.
	files map[string]*File

	// Lock protecting the above.
	filesLock sync.RWMutex

	// Our file map priority queue,
	// used for closing unused file descriptors.
	lru []*File

	// Lock protecting the above.
	lruLock sync.Mutex

	// Our file root.
	root *File

	// Our next fileid.
	Fileid uint64 `json:"fileid"`

	// Our file descriptor limits.
	Fdlimit uint `json:"fdlimit"`
}

func (fs *Fs) error(buf Buffer, tag uint16, err error) error {

	// Was there an error?
	// If so, encode it properly.
	int_err, ok := err.(*Error)
	if ok {
		// We have a specific error code.
		return PackRerror(buf, tag, int_err.Err, int_err.Errornum, fs.Dotu)
	}

	// This is a generic error.
	return PackRerror(buf, tag, err.Error(), uint32(syscall.EIO), fs.Dotu)
}

func (fs *Fs) wait(tag uint16) {
	fs.fidLock.RLock()
	for fs.Reqs[tag] {
		fs.fidCond.Wait()
	}
	fs.fidLock.RUnlock()
}

func (fs *Fs) clear(tag uint16) {
	fs.fidLock.Lock()
	delete(fs.Reqs, tag)
	fs.fidCond.Broadcast()
	fs.fidLock.Unlock()
}

func (fs *Fs) Handle(req Buffer, resp Buffer, debug bool) error {

	var fid *Fid
	var afid *Fid

	// Unpack our message.
	fcall, err := Unpack(req, fs.Dotu)
	if err != nil {
		log.Printf("9pfs serialization error?")
		err = fs.error(resp, NOTAG, err)
		goto done
	}

	if debug {
		log.Printf("req: Fcall -> %s", fcall.String())
	}

	// Save our tag.
	if fcall.Tag != NOTAG {
		fs.fidLock.Lock()
		_, ok := fs.Reqs[fcall.Tag]
		if ok {
			fs.fidLock.Unlock()
			err = fs.error(resp, fcall.Tag, Einuse)
			goto done
		}
		fs.Reqs[fcall.Tag] = true
		defer fs.clear(fcall.Tag)
		fs.fidLock.Unlock()
	}

	// Is there an fid here?
	if fcall.Fid != NOFID &&
		fcall.Type != Tattach &&
		fcall.Type != Tauth &&
		fcall.Type != Tversion {

		fid = fs.GetFid(fcall.Fid)
		if fid == nil {
			err = fs.error(resp, fcall.Tag, Eunknownfid)
			goto done
		}
		defer fid.DecRef(fs)
	}
	if fcall.Afid != NOFID &&
		fcall.Type == Tattach {

		afid = fs.GetFid(fcall.Afid)
		if afid == nil {
			err = fs.error(resp, fcall.Tag, Eunknownfid)
			goto done
		}
		defer afid.DecRef(fs)
	}

	switch fcall.Type {
	case Tversion:
		var msize uint32
		var dotu bool
		var version string
		msize, dotu, version, err = fs.version(fcall.Msize, fcall.Version)
		if err == nil {
			err = PackRversion(resp, fcall.Tag, msize, version)
		}
		if err == nil {
			fs.versionPost(msize, dotu)
		}

	case Tauth:
		var qid *Qid
		qid, err = fs.auth(fcall.Afid, fcall.Uname, fcall.Aname, fcall.Unamenum)
		if err == nil {
			err = PackRauth(resp, fcall.Tag, qid)
		}

	case Tattach:
		var qid *Qid
		qid, err = fs.attach(fcall.Fid, afid, fcall.Uname, fcall.Aname, fcall.Unamenum)
		if err == nil {
			err = PackRattach(resp, fcall.Tag, qid)
		}

	case Tflush:
		fs.wait(fcall.Oldtag)
		err = PackRflush(resp, fcall.Tag)

	case Twalk:
		var wqids []Qid
		wqids, err = fs.walk(fid, fcall.Newfid, fcall.Wname)
		if err == nil {
			err = PackRwalk(resp, fcall.Tag, wqids)
		}

	case Topen:
		var qid *Qid
		var iounit uint32
		qid, iounit, err = fs.open(fid, fcall.Mode)
		if err == nil {
			err = PackRopen(resp, fcall.Tag, qid, iounit)
		}
		if err == nil {
			fs.openPost(fid, fcall.Mode)
		}

	case Tcreate:
		var file *File
		var qid *Qid
		var iounit uint32
		file, qid, iounit, err = fs.create(fid, fcall.Name, fcall.Perm, fcall.Mode, fcall.Ext)
		if err == nil {
			err = PackRcreate(resp, fcall.Tag, qid, iounit)
		}
		if err == nil {
			fs.createPost(fid, file, fcall.Name, fcall.Mode)
		} else {
			fs.createFail(fid, file, fcall.Name, fcall.Mode)
		}

	case Tread:
		// NOTE: This is a somewhat ugly special case.
		// Because of the way the interface is designed here,
		// we have to specially check for directory reads.
		if fid == nil {
			err = Eunknownfid

		} else if fid.file.Qid.Type&QTDIR != 0 {

			var children []*Dir
			var count int
			var written int

			children, err = fs.readDir(fid, int64(fcall.Offset), int(fcall.Count))
			if err == nil {
				// Pack with no count.
				err = PackRread(resp, fcall.Tag, 0)
			}
			if err == nil {
				// Pack the given entries.
				for count < len(children) {
					if debug {
						log.Printf(
							"fid %x child[%d] -> %s",
							fcall.Fid,
							count,
							children[count].Name)
					}
					packed := pstat(resp, children[count], fs.Dotu)
					if resp.WriteLeft() >= 0 &&
						written+packed < int(fcall.Count) {
						count += 1
						written += packed
					} else {
						// This one didn't count.
						// We weren't able to fit the entire
						// directory encoded here, so do it next time.
						break
					}
				}
				// Repack with the appropriate count.
				err = PackRread(resp, fcall.Tag, uint32(written))
			}
			if err == nil {
				fs.readDirPost(fid, uint32(written), count)
			}

		} else {

			var fd int
			var length int

			fd, err = fs.readFile(fid, int64(fcall.Offset), int(fcall.Count))
			if err == nil {
				// Pack with no count.
				err = PackRread(resp, fcall.Tag, 0)
			}
			if err == nil {
				// Perform the actual read.
				length, err = resp.ReadFromFd(fd, int64(fcall.Offset), int(fcall.Count))
				if err == nil {
					// Repack with the appropriate count.
					err = PackRread(resp, fcall.Tag, uint32(length))
				}
			}
			if err == nil {
				fs.readFilePost(fid, uint32(length))
			} else {
				fs.readFileFail(fid, uint32(length))
			}
		}

	case Twrite:
		// NOTE: Ugly as per above.
		// No writes should happen on directories.

		if fid == nil {
			err = Eunknownfid

		} else if fid.file.Qid.Type&QTDIR != 0 {
			err = Ebaduse

		} else {

			var fd int
			var length int

			fd, err = fs.writeFile(fid, int64(fcall.Offset), int(fcall.Count))
			if err == nil {
				err = PackRwrite(resp, fcall.Tag, 0)
			}
			if err == nil {
				// Perform the actual write.
				length, err = req.WriteToFd(fd, int64(fcall.Offset), int(fcall.Count))
				if err == nil {
					// Repack with the appropriate count.
					err = PackRwrite(resp, fcall.Tag, uint32(length))
				}
			}
			if err == nil {
				fs.writeFilePost(fid, uint32(length))
			} else {
				fs.writeFileFail(fid, uint32(length))
			}
		}

	case Tclunk:
		err = fs.clunk(fid)
		if err == nil {
			err = PackRclunk(resp, fcall.Tag)
		}
		if err == nil {
			fs.clunkPost(fid)
		}

	case Tremove:
		err = fs.remove(fid)
		if err == nil {
			err = PackRremove(resp, fcall.Tag)
		}
		if err == nil {
			err = fs.removePost(fid)
		}

	case Tstat:
		var dir *Dir
		dir, err = fs.stat(fid)
		if err == nil {
			err = PackRstat(resp, fcall.Tag, dir, fs.Dotu)
		}

	case Twstat:
		var dir *Dir
		dir, err = fs.wstat(fid, &fcall.Dir)
		if err == nil {
			err = PackRwstat(resp, fcall.Tag)
		}
		if err == nil {
			err = fs.wstatPost(fid, dir, &fcall.Dir)
		} else {
			fs.wstatFail(fid, dir, &fcall.Dir)
		}

	default:
		err = InvalidMessage
	}

	if err != nil {
		// An error? Re-encode.
		err = fs.error(resp, fcall.Tag, err)
		goto done
	}

done:
	if debug {
		resp.ReadRewind()
		rcall, err := Unpack(resp, fs.Dotu)
		if err != nil {
			log.Printf("9pfs response error? req: Fcall -> %s", fcall.String())
			return err
		}

		// Print our result.
		log.Printf("resp: Fcall <- %s", rcall.String())
	}

	if debug {
		fs.fidLock.Lock()
		fs.filesLock.Lock()
		for fidno, fid := range fs.Pool {
			log.Printf(
				"  fid %x: %d refs (%x)",
				fidno, fid.refs, fid.file.Qid.Path)
		}
		for path, file := range fs.files {
			log.Printf(
				"  file %x: %s => %d refs",
				file.Qid.Path, path, file.refs)
		}
		fs.fidLock.Unlock()
		fs.filesLock.Unlock()
	}

	// All good.
	return nil
}

func (fs *Fs) Init() error {
	fs.Read = make(map[string][]string)
	fs.Write = make(map[string]string)
	fs.Dotu = true
	fs.Pool = make(map[uint32]*Fid)
	fs.Reqs = make(map[uint16]bool)
	fs.fidCond = sync.NewCond(&fs.fidLock)
	fs.files = make(map[string]*File)
	fs.lru = make([]*File, 0, 0)

	return nil
}

func (fs *Fs) Attach() error {

	fs.fidLock.Lock()
	defer fs.fidLock.Unlock()

	// Load the root.
	var err error
	fs.root, err = fs.lookup("/")
	if err != nil {
		return err
	}

	if fs.Fdlimit == 0 {
		// Figure out our active limit (1/2 open limit).
		// We use 1/2 because control connections, tap devices,
		// disks, etc. all need file descriptors. Note that we
		// also explicitly handle running out of file descriptors,
		// but this gives us an open bound to leave room for the
		// rest of the system (because pieces don't always handle
		// an EMFILE or ENFILE appropriately).
		var rlim syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
		if err != nil {
			return err
		}
		fs.Fdlimit = uint(rlim.Cur) / 2
	}

	return nil
}

func (reqs *Reqlist) MarshalJSON() ([]byte, error) {

	// Create an array.
	reqlist := make([]uint16, 0, len(*reqs))
	for req, _ := range *reqs {
		reqlist = append(reqlist, req)
	}

	// Marshal as an array.
	return json.Marshal(reqlist)
}

func (reqs *Reqlist) UnmarshalJSON(data []byte) error {

	// Unmarshal as an array.
	reqlist := make([]uint16, 0, len(*reqs))
	err := json.Unmarshal(data, &reqlist)
	if err != nil {
		return err
	}

	// Load all elements.
	for _, req := range reqlist {
		(*reqs)[req] = true
	}

	return nil
}
