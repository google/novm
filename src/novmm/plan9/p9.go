// Original file Copyright 2009 The Go9p Authors. All rights reserved.
// Full license available in licenses/go9p.
//
// Modifications Copyright 2014 Google Inc. All rights reserved.
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

// 9P2000 message types
const (
	Tversion = 100 + iota
	Rversion
	Tauth
	Rauth
	Tattach
	Rattach
	Terror
	Rerror
	Tflush
	Rflush
	Twalk
	Rwalk
	Topen
	Ropen
	Tcreate
	Rcreate
	Tread
	Rread
	Twrite
	Rwrite
	Tclunk
	Rclunk
	Tremove
	Rremove
	Tstat
	Rstat
	Twstat
	Rwstat
	Tlast
)

const (
	MSIZE   = 8192 + IOHDRSZ // default message size (8192+IOHdrSz)
	IOHDRSZ = 24             // the non-data size of the Twrite messages
	PORT    = 564            // default port for 9P file servers
)

// Qid types
const (
	QTDIR     = 0x80 // directories
	QTAPPEND  = 0x40 // append only files
	QTEXCL    = 0x20 // exclusive use files
	QTMOUNT   = 0x10 // mounted channel
	QTAUTH    = 0x08 // authentication file
	QTTMP     = 0x04 // non-backed-up file
	QTSYMLINK = 0x02 // symbolic link (Unix, 9P2000.u)
	QTLINK    = 0x01 // hard link (Unix, 9P2000.u)
	QTFILE    = 0x00
)

// Flags for the mode field in Topen and Tcreate messages
const (
	OREAD   = 0  // open read-only
	OWRITE  = 1  // open write-only
	ORDWR   = 2  // open read-write
	OEXEC   = 3  // execute (== read but check execute permission)
	OTRUNC  = 16 // or'ed in (except for exec), truncate file first
	OCEXEC  = 32 // or'ed in, close on exec
	ORCLOSE = 64 // or'ed in, remove on close
)

// File modes
const (
	DMDIR       = 0x80000000 // mode bit for directories
	DMAPPEND    = 0x40000000 // mode bit for append only files
	DMEXCL      = 0x20000000 // mode bit for exclusive use files
	DMMOUNT     = 0x10000000 // mode bit for mounted channel
	DMAUTH      = 0x08000000 // mode bit for authentication file
	DMTMP       = 0x04000000 // mode bit for non-backed-up file
	DMSYMLINK   = 0x02000000 // mode bit for symbolic link (Unix, 9P2000.u)
	DMLINK      = 0x01000000 // mode bit for hard link (Unix, 9P2000.u)
	DMDEVICE    = 0x00800000 // mode bit for device file (Unix, 9P2000.u)
	DMNAMEDPIPE = 0x00200000 // mode bit for named pipe (Unix, 9P2000.u)
	DMSOCKET    = 0x00100000 // mode bit for socket (Unix, 9P2000.u)
	DMSETUID    = 0x00080000 // mode bit for setuid (Unix, 9P2000.u)
	DMSETGID    = 0x00040000 // mode bit for setgid (Unix, 9P2000.u)
	DMREAD      = 0x4        // mode bit for read permission
	DMWRITE     = 0x2        // mode bit for write permission
	DMEXEC      = 0x1        // mode bit for execute permission
)

const (
	NOTAG uint16 = 0xFFFF     // no tag specified
	NOFID uint32 = 0xFFFFFFFF // no fid specified
	NOUID uint32 = 0xFFFFFFFF // no uid specified
)

// Error values
const (
	EPERM   = 1
	ENOENT  = 2
	EIO     = 5
	EEXIST  = 17
	ENOTDIR = 20
	EINVAL  = 22
)

// Error represents a 9P2000 (and 9P2000.u) error.
type Error struct {
	Err      string // textual representation of the error
	Errornum uint32 // numeric representation of the error (9P2000.u)
}

// File identifier.
type Qid struct {
	Type    uint8  // type of the file (high 8 bits of the mode)
	Version uint32 // version number for the path
	Path    uint64 // server's unique identification of the file
}

// Dir describes a file.
type Dir struct {
	Type   uint16
	Dev    uint32
	Qid           // file's Qid
	Mode   uint32 // permissions and flags
	Atime  uint32 // last access time in seconds
	Mtime  uint32 // last modified time in seconds
	Length uint64 // file length in bytes
	Name   string // file name
	Uid    string // owner name
	Gid    string // group name
	Muid   string // name of the last user that modified the file

	// 9P2000.u extension
	Ext     string // special file's descriptor
	Uidnum  uint32 // owner ID
	Gidnum  uint32 // group ID
	Muidnum uint32 // ID of the last user that modified the file
}

// Fcall represents a 9P2000 message.
type Fcall struct {
	Size    uint32   // size of the message
	Type    uint8    // message type
	Fid     uint32   // file identifier
	Tag     uint16   // message tag
	Msize   uint32   // maximum message size (used by Tversion, Rversion)
	Version string   // protocol version (used by Tversion, Rversion)
	Oldtag  uint16   // tag of the message to flush (used by Tflush)
	Error   string   // error (used by Rerror)
	Qid              // file Qid (used by Rauth, Rattach, Ropen, Rcreate)
	Iounit  uint32   // maximum bytes read without breaking in multiple messages (used by Ropen, Rcreate)
	Afid    uint32   // authentication fid (used by Tauth, Tattach)
	Uname   string   // user name (used by Tauth, Tattach)
	Aname   string   // attach name (used by Tauth, Tattach)
	Perm    uint32   // file permission (mode) (used by Tcreate)
	Name    string   // file name (used by Tcreate)
	Mode    uint8    // open mode (used by Topen, Tcreate)
	Newfid  uint32   // the fid that represents the file walked to (used by Twalk)
	Wname   []string // list of names to walk (used by Twalk)
	Wqid    []Qid    // list of Qids for the walked files (used by Rwalk)
	Offset  uint64   // offset in the file to read/write from/to (used by Tread, Twrite)
	Count   uint32   // number of bytes read/written (used by Tread, Rread, Twrite, Rwrite)
	Dir              // file description (used by Rstat, Twstat)

	/* 9P2000.u extensions */
	Errornum uint32 // error code, 9P2000.u only (used by Rerror)
	Ext      string // special file description, 9P2000.u only (used by Tcreate)
	Unamenum uint32 // user ID, 9P2000.u only (used by Tauth, Tattach)
}

// minimum size of a 9P2000 message for a type
var minFcsize = [...]uint32{
	6,  /* Tversion msize[4] version[s] */
	6,  /* Rversion msize[4] version[s] */
	8,  /* Tauth fid[4] uname[s] aname[s] */
	13, /* Rauth aqid[13] */
	12, /* Tattach fid[4] afid[4] uname[s] aname[s] */
	13, /* Rattach qid[13] */
	0,  /* Terror */
	2,  /* Rerror ename[s] (ecode[4]) */
	2,  /* Tflush oldtag[2] */
	0,  /* Rflush */
	10, /* Twalk fid[4] newfid[4] nwname[2] */
	2,  /* Rwalk nwqid[2] */
	5,  /* Topen fid[4] mode[1] */
	17, /* Ropen qid[13] iounit[4] */
	11, /* Tcreate fid[4] name[s] perm[4] mode[1] */
	17, /* Rcreate qid[13] iounit[4] */
	16, /* Tread fid[4] offset[8] count[4] */
	4,  /* Rread count[4] */
	16, /* Twrite fid[4] offset[8] count[4] */
	4,  /* Rwrite count[4] */
	4,  /* Tclunk fid[4] */
	0,  /* Rclunk */
	4,  /* Tremove fid[4] */
	0,  /* Rremove */
	4,  /* Tstat fid[4] */
	4,  /* Rstat stat[n] */
	8,  /* Twstat fid[4] stat[n] */
	0,  /* Rwstat */
	20, /* Tbread fileid[8] offset[8] count[4] */
	4,  /* Rbread count[4] */
	20, /* Tbwrite fileid[8] offset[8] count[4] */
	4,  /* Rbwrite count[4] */
	16, /* Tbtrunc fileid[8] offset[8] */
	0,  /* Rbtrunc */
}

// minimum size of a 9P2000.u message for a type
var minFcusize = [...]uint32{
	6,  /* Tversion msize[4] version[s] */
	6,  /* Rversion msize[4] version[s] */
	12, /* Tauth fid[4] uname[s] aname[s] */
	13, /* Rauth aqid[13] */
	16, /* Tattach fid[4] afid[4] uname[s] aname[s] */
	13, /* Rattach qid[13] */
	0,  /* Terror */
	6,  /* Rerror ename[s] (ecode[4]) */
	2,  /* Tflush oldtag[2] */
	0,  /* Rflush */
	10, /* Twalk fid[4] newfid[4] nwname[2] */
	2,  /* Rwalk nwqid[2] */
	5,  /* Topen fid[4] mode[1] */
	17, /* Ropen qid[13] iounit[4] */
	13, /* Tcreate fid[4] name[s] perm[4] mode[1] */
	17, /* Rcreate qid[13] iounit[4] */
	16, /* Tread fid[4] offset[8] count[4] */
	4,  /* Rread count[4] */
	16, /* Twrite fid[4] offset[8] count[4] */
	4,  /* Rwrite count[4] */
	4,  /* Tclunk fid[4] */
	0,  /* Rclunk */
	4,  /* Tremove fid[4] */
	0,  /* Rremove */
	4,  /* Tstat fid[4] */
	4,  /* Rstat stat[n] */
	8,  /* Twstat fid[4] stat[n] */
	20, /* Tbread fileid[8] offset[8] count[4] */
	4,  /* Rbread count[4] */
	20, /* Tbwrite fileid[8] offset[8] count[4] */
	4,  /* Rbwrite count[4] */
	16, /* Tbtrunc fileid[8] offset[8] */
	0,  /* Rbtrunc */
}
