// Copyright 2009 The Go9p Authors.  All rights reserved.
// Copyright 2013 Adin Scannell.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the licenses/go9p file.
package plan9

import (
    "log"
    "sync"
    "syscall"
)

type Fs struct {

    // The read mappings.
    Read map[string][]string `json:"read"`

    // The write mappings.
    Write map[string]string `json:"write"`

    // are we speaking 9P2000.u?
    Dotu bool `json:"dotu"`

    // open FiDs.
    Fidpool map[uint32]*Fid `json:"fidpool"`

    // requests outstanding.
    Reqs map[uint16]bool `json:"reqs"`

    // Lock protecting the above.
    fidLock sync.RWMutex
    fidCond *sync.Cond

    // Our open files.
    files map[string]*File

    // Lock protecting the above.
    fileLock sync.RWMutex

    // Our file root.
    root *File

    // Our next fileid.
    Fileid uint64 `json:"fileid"`
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
                            "fid %d child[%d] -> %s",
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
        err = fs.wstat(fid, &fcall.Dir)
        if err == nil {
            err = PackRwstat(resp, fcall.Tag)
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
        fs.fileLock.Lock()
        for fidno, fid := range fs.Fidpool {
            log.Printf(
                "  fidno %x: %d refs (%s)",
                fidno, fid.Refs, fid.file)
        }
        for path, file := range fs.files {
            log.Printf(
                "  file %x: %s => %d refs",
                file.Qid.Path, path, file.refs)
        }
        fs.fidLock.Unlock()
        fs.fileLock.Unlock()
    }

    // All good.
    return nil
}

func (fs *Fs) Init() error {
    fs.Read = make(map[string][]string)
    fs.Write = make(map[string]string)
    fs.Dotu = true
    fs.Fidpool = make(map[uint32]*Fid)
    fs.Reqs = make(map[uint16]bool)
    fs.fidCond = sync.NewCond(&fs.fidLock)
    fs.files = make(map[string]*File)
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

    // Restore all our Fids.
    for _, fid := range fs.Fidpool {
        fid.file, err = fs.lookup(fid.Path)
        if err != nil {
            return err
        }
    }

    return nil
}
