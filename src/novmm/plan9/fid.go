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
	"sync/atomic"
)

type Fid struct {
	// The associated Fid.
	Fid uint32 `json:"fid"`

	// Runtime references.
	refs int32 `json:"-"`

	// The associated path.
	Path string `json:"path"`

	// True if the Fid is opened.
	Opened bool `json:"opened"`

	// Open mode (O* flags).
	Omode uint8 `json:"omode"`

	// If directory, next valid read position.
	Diroffset uint64 `json:"diroffset"`

	// If directory, list of children (reset by read(offset=0).
	Direntries []*Dir `json:"direntries"`

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

//
// Our collection of Fids.
//
// This is a special type to handle marshal/unmarshal.
type Fidpool map[uint32]*Fid

func (fs *Fs) GetFid(fidno uint32) *Fid {

	fs.fidLock.RLock()
	fid, present := fs.Pool[fidno]
	if present {
		atomic.AddInt32(&fid.refs, 1)
	}
	fs.fidLock.RUnlock()

	return fid
}

func (fs *Fs) NewFid(
	fidno uint32,
	path string,
	file *File) (*Fid, error) {

	fs.fidLock.Lock()
	_, present := fs.Pool[fidno]
	if present {
		fs.fidLock.Unlock()
		return nil, Einuse
	}

	fid := new(Fid)
	fid.Fid = fidno
	fid.refs = 1
	fid.Path = path
	fid.file = file

	fs.Pool[fidno] = fid
	fs.fidLock.Unlock()

	return fid, nil
}

func (fid *Fid) DecRef(fs *Fs) {

	new_refs := atomic.AddInt32(&fid.refs, -1)

	if new_refs == 0 {
		fs.fidLock.Lock()
		if fid.refs != 0 {
			// Race condition caught.
			fs.fidLock.Unlock()
			return
		}

		// Remove this fid.
		delete(fs.Pool, fid.Fid)
		fs.fidLock.Unlock()

		// Lose our file reference.
		fid.file.DecRef(fs, fid.Path)
	}
}

func (fidpool *Fidpool) MarshalJSON() ([]byte, error) {

	// Create an array.
	fids := make([]*Fid, 0, len(*fidpool))
	for _, fid := range *fidpool {
		fids = append(fids, fid)
	}

	// Marshal as an array.
	return json.Marshal(fids)
}

func (fidpool *Fidpool) UnmarshalJSON(data []byte) error {

	// Unmarshal as an array.
	fids := make([]*Fid, 0, 0)
	err := json.Unmarshal(data, &fids)
	if err != nil {
		return err
	}

	// Load all elements.
	for _, fid := range fids {
		(*fidpool)[fid.Fid] = fid
	}

	return nil
}
