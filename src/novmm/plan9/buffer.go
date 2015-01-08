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

type Buffer interface {
	ReadLeft() int
	WriteLeft() int

	ReadRewind()
	WriteRewind()

	Read8() uint8
	Read16() uint16
	Read32() uint32
	Read64() uint64
	ReadBytes(length int) []byte
	ReadString() string

	Write8(value uint8)
	Write16(value uint16)
	Write32(value uint32)
	Write64(value uint64)
	WriteBytes(value []byte)
	WriteString(value string)

	ReadFromFd(fd int, offset int64, length int) (int, error)
	WriteToFd(fd int, offset int64, length int) (int, error)
}

func gqid(buf Buffer, qid *Qid) {
	qid.Type = buf.Read8()
	qid.Version = buf.Read32()
	qid.Path = buf.Read64()
}

func gstat(buf Buffer, d *Dir, dotu bool) {
	buf.Read16() // Read length.
	d.Type = buf.Read16()
	d.Dev = buf.Read32()
	gqid(buf, &d.Qid)
	d.Mode = buf.Read32()
	d.Atime = buf.Read32()
	d.Mtime = buf.Read32()
	d.Length = buf.Read64()
	d.Name = buf.ReadString()
	d.Uid = buf.ReadString()
	d.Gid = buf.ReadString()
	d.Muid = buf.ReadString()
	if dotu {
		d.Ext = buf.ReadString()
		d.Uidnum = buf.Read32()
		d.Gidnum = buf.Read32()
		d.Muidnum = buf.Read32()
	} else {
		d.Uidnum = NOUID
		d.Gidnum = NOUID
		d.Muidnum = NOUID
	}
}

func statsz(d *Dir, dotu bool) int {
	sz := 2 + 2 + 4 + 13 + 4 + 4 + 4 + 8 + 2 + 2 + 2 + 2 +
		len(d.Name) + len(d.Uid) + len(d.Gid) + len(d.Muid)
	if dotu {
		sz += 2 + 4 + 4 + 4 + len(d.Ext)
	}

	return sz
}

func pqid(buf Buffer, qid *Qid) {
	buf.Write8(qid.Type)
	buf.Write32(qid.Version)
	buf.Write64(qid.Path)
}

func pstat(buf Buffer, d *Dir, dotu bool) int {
	sz := statsz(d, dotu)
	buf.Write16(uint16(sz - 2))
	buf.Write16(d.Type)
	buf.Write32(d.Dev)
	pqid(buf, &d.Qid)
	buf.Write32(d.Mode)
	buf.Write32(d.Atime)
	buf.Write32(d.Mtime)
	buf.Write64(d.Length)
	buf.WriteString(d.Name)
	buf.WriteString(d.Uid)
	buf.WriteString(d.Gid)
	buf.WriteString(d.Muid)
	if dotu {
		buf.WriteString(d.Ext)
		buf.Write32(d.Uidnum)
		buf.Write32(d.Gidnum)
		buf.Write32(d.Muidnum)
	}
	return sz
}
