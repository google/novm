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

import "fmt"

func permToString(perm uint32) string {
	ret := ""

	if perm&DMDIR != 0 {
		ret += "d"
	}
	if perm&DMAPPEND != 0 {
		ret += "a"
	}
	if perm&DMAUTH != 0 {
		ret += "A"
	}
	if perm&DMEXCL != 0 {
		ret += "l"
	}
	if perm&DMTMP != 0 {
		ret += "t"
	}
	if perm&DMDEVICE != 0 {
		ret += "D"
	}
	if perm&DMSOCKET != 0 {
		ret += "S"
	}
	if perm&DMNAMEDPIPE != 0 {
		ret += "P"
	}
	if perm&DMSYMLINK != 0 {
		ret += "L"
	}

	ret += fmt.Sprintf("%o", perm&0777)
	return ret
}

func (qid *Qid) String() string {
	b := ""
	if qid.Type&QTDIR != 0 {
		b += "d"
	}
	if qid.Type&QTAPPEND != 0 {
		b += "a"
	}
	if qid.Type&QTAUTH != 0 {
		b += "A"
	}
	if qid.Type&QTEXCL != 0 {
		b += "l"
	}
	if qid.Type&QTTMP != 0 {
		b += "t"
	}
	if qid.Type&QTSYMLINK != 0 {
		b += "L"
	}

	return fmt.Sprintf("(%x %x '%s')", qid.Path, qid.Version, b)
}

func (d *Dir) String() string {
	ret := fmt.Sprintf("'%s' '%s' '%s' '%s' q ", d.Name, d.Uid, d.Gid, d.Muid)
	ret += d.Qid.String() + " m " + permToString(d.Mode)
	ret += fmt.Sprintf(" at %d mt %d l %d t %d d %d", d.Atime, d.Mtime,
		d.Length, d.Type, d.Dev)

	/* dotu ? */
	ret += " ext " + d.Ext

	return ret
}

func (fc *Fcall) String() string {
	ret := ""

	switch fc.Type {
	case Tversion:
		ret = fmt.Sprintf(
			"Tversion tag %d msize %d version '%s'",
			fc.Tag, fc.Msize, fc.Version)
	case Rversion:
		ret = fmt.Sprintf(
			"Rversion tag %d msize %d version '%s'",
			fc.Tag, fc.Msize, fc.Version)
	case Tauth:
		ret = fmt.Sprintf(
			"Tauth tag %d afid %x uname '%s' nuname %d aname '%s'",
			fc.Tag, fc.Afid, fc.Uname, fc.Unamenum, fc.Aname)
	case Rauth:
		ret = fmt.Sprintf(
			"Rauth tag %d aqid %v", fc.Tag, &fc.Qid)
	case Rattach:
		ret = fmt.Sprintf(
			"Rattach tag %d aqid %v", fc.Tag, &fc.Qid)
	case Tattach:
		ret = fmt.Sprintf(
			"Tattach tag %d fid %x afid %x uname '%s' nuname %d aname '%s'",
			fc.Tag, fc.Fid, fc.Afid, fc.Uname, fc.Unamenum, fc.Aname)
	case Tflush:
		ret = fmt.Sprintf(
			"Tflush tag %d oldtag %d",
			fc.Tag, fc.Oldtag)
	case Rerror:
		ret = fmt.Sprintf(
			"Rerror tag %d ename '%s' ecode %d",
			fc.Tag, fc.Error, fc.Errornum)
	case Twalk:
		ret = fmt.Sprintf(
			"Twalk tag %d fid %x newfid %x ",
			fc.Tag, fc.Fid, fc.Newfid)
		for i := 0; i < len(fc.Wname); i++ {
			ret += fmt.Sprintf("%d:'%s' ", i, fc.Wname[i])
		}
	case Rwalk:
		ret = fmt.Sprintf("Rwalk tag %d ", fc.Tag)
		for i := 0; i < len(fc.Wqid); i++ {
			ret += fmt.Sprintf("%v ", &fc.Wqid[i])
		}
	case Topen:
		ret = fmt.Sprintf(
			"Topen tag %d fid %x mode %x",
			fc.Tag, fc.Fid, fc.Mode)
	case Ropen:
		ret = fmt.Sprintf(
			"Ropen tag %d qid %v iounit %d",
			fc.Tag, &fc.Qid, fc.Iounit)
	case Rcreate:
		ret = fmt.Sprintf(
			"Rcreate tag %d qid %v iounit %d",
			fc.Tag, &fc.Qid, fc.Iounit)
	case Tcreate:
		ret = fmt.Sprintf(
			"Tcreate tag %d fid %x name '%s' perm %s mode %x",
			fc.Tag, fc.Fid, fc.Name, permToString(fc.Perm), fc.Mode)
	case Tread:
		ret = fmt.Sprintf(
			"Tread tag %d fid %x offset %d count %d",
			fc.Tag, fc.Fid, fc.Offset, fc.Count)
	case Rread:
		ret = fmt.Sprintf(
			"Rread tag %d count %d",
			fc.Tag, fc.Count)
	case Twrite:
		ret = fmt.Sprintf(
			"Twrite tag %d fid %x offset %d count %d",
			fc.Tag, fc.Fid, fc.Offset, fc.Count)
	case Rwrite:
		ret = fmt.Sprintf(
			"Rwrite tag %d count %d",
			fc.Tag, fc.Count)
	case Tclunk:
		ret = fmt.Sprintf(
			"Tclunk tag %d fid %x",
			fc.Tag, fc.Fid)
	case Rclunk:
		ret = fmt.Sprintf(
			"Rclunk tag %d",
			fc.Tag)
	case Tremove:
		ret = fmt.Sprintf(
			"Tremove tag %d fid %x",
			fc.Tag, fc.Fid)
	case Tstat:
		ret = fmt.Sprintf(
			"Tstat tag %d fid %x",
			fc.Tag, fc.Fid)
	case Rstat:
		ret = fmt.Sprintf(
			"Rstat tag %d st (%v)",
			fc.Tag, &fc.Dir)
	case Twstat:
		ret = fmt.Sprintf(
			"Twstat tag %d fid %x st (%v)",
			fc.Tag, fc.Fid, &fc.Dir)
	case Rflush:
		ret = fmt.Sprintf(
			"Rflush tag %d",
			fc.Tag)
	case Rremove:
		ret = fmt.Sprintf(
			"Rremove tag %d",
			fc.Tag)
	case Rwstat:
		ret = fmt.Sprintf(
			"Rwstat tag %d",
			fc.Tag)
	default:
		ret = fmt.Sprintf(
			"invalid call: %d",
			fc.Type)
	}

	return ret
}

func (err *Error) Error() string {
	if err != nil {
		return fmt.Sprintf("%s: %d", err.Err, err.Errornum)
	}

	return ""
}
