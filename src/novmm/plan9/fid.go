// Copyright 2009 The Go9p Authors.  All rights reserved.
// Copyright 2013 Adin Scannell.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the licenses/go9p file.
package plan9

import (
    "sync/atomic"
)

type Fid struct {
    // The associated Fid.
    Fid uint32 `json:"fid"`

    // Runtime references.
    Refs int32 `json:"refs"`

    // The associated path.
    Path string `json:"path"`

    // True if the Fid is opened.
    Opened bool `json:"opened"`

    // Open mode (O* flags).
    Omode uint8 `json:"omode"`

    // If directory, next valid read position.
    Diroffset uint64 `json:"diroffset"`

    // If directory, list of children (reset by read(offset=0).
    direntries []*Dir

    // The associated file.
    //
    // This is looked up from the filesystem map
    // when the new Fid is created. Note that multiple
    // Fids may refer to the same file object, and
    // they are refcounted separately.
    //
    // This is not serialized like the rest of the Fid.
    // When the Fs is loaded, the Init() method is called,
    // which will reinitialize all the File objects.
    file *File
}

func (fs *Fs) GetFid(fidno uint32) *Fid {

    fs.fidLock.RLock()
    fid, present := fs.Fidpool[fidno]
    if present {
        atomic.AddInt32(&fid.Refs, 1)
    }
    fs.fidLock.RUnlock()

    return fid
}

func (fs *Fs) NewFid(
    fidno uint32,
    path string,
    file *File) (*Fid, error) {

    fs.fidLock.Lock()
    _, present := fs.Fidpool[fidno]
    if present {
        fs.fidLock.Unlock()
        return nil, Einuse
    }

    fid := new(Fid)
    fid.Fid = fidno
    fid.Refs = 1
    fid.Path = path
    fid.file = file

    fs.Fidpool[fidno] = fid
    fs.fidLock.Unlock()

    return fid, nil
}

func (fid *Fid) DecRef(fs *Fs) {

    new_refs := atomic.AddInt32(&fid.Refs, -1)

    if new_refs == 0 {
        fs.fidLock.Lock()
        if fid.Refs != 0 {
            // Race condition caught.
            fs.fidLock.Unlock()
            return
        }

        // Remove this fid.
        delete(fs.Fidpool, fid.Fid)
        fs.fidLock.Unlock()

        // Lose our file reference.
        fid.file.DecRef(fs, fid.Path)
    }
}
