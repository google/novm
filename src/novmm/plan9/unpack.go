// Copyright 2009 The Go9p Authors.  All rights reserved.
// Copyright 2013 Adin Scannell.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the licenses/go9p file.
package plan9

import (
    "log"
)

// Creates a Fcall value from the on-the-wire representation. If
// dotu is true, reads 9P2000.u messages. Returns the unpacked message,
// error and how many bytes from the buffer were used by the message.
func Unpack(
    buf Buffer,
    dotu bool) (*Fcall, error) {

    // Enough for a header?
    if buf.ReadLeft() < 7 {
        log.Printf("buffer smaller than header?")
        return nil, BufferInsufficient
    }

    fc := new(Fcall)
    fc.Fid = NOFID
    fc.Afid = NOFID
    fc.Newfid = NOFID

    fc.Size = buf.Read32()
    fc.Type = buf.Read8()
    fc.Tag = buf.Read16()

    // Sanity check the size.
    if int(fc.Size)-7 > buf.ReadLeft() || fc.Size < 7 {
        log.Printf("size is smaller than header?")
        return nil, BufferInsufficient
    }

    if fc.Type < Tversion || fc.Type >= Tlast {
        return nil, InvalidMessage
    }

    var sz uint32
    if dotu {
        sz = minFcsize[fc.Type-Tversion]
    } else {
        sz = minFcusize[fc.Type-Tversion]
    }

    if fc.Size < sz {
        log.Printf("buffer doesn't match size?")
        return nil, BufferInsufficient
    }

    var err error
    switch fc.Type {
    case Tversion, Rversion:
        fc.Msize = buf.Read32()
        fc.Version = buf.ReadString()

    case Tauth:
        fc.Afid = buf.Read32()
        fc.Uname = buf.ReadString()
        fc.Aname = buf.ReadString()

        if dotu {
            if buf.ReadLeft() > 0 {
                fc.Unamenum = buf.Read32()
            } else {
                fc.Unamenum = NOUID
            }
        } else {
            fc.Unamenum = NOUID
        }

    case Rauth, Rattach:
        gqid(buf, &fc.Qid)

    case Tflush:
        fc.Oldtag = buf.Read16()

    case Tattach:
        fc.Fid = buf.Read32()
        fc.Afid = buf.Read32()
        fc.Uname = buf.ReadString()
        fc.Aname = buf.ReadString()

        if dotu {
            if buf.ReadLeft() > 0 {
                fc.Unamenum = buf.Read32()
            } else {
                fc.Unamenum = NOUID
            }
        }

    case Rerror:
        fc.Error = buf.ReadString()
        if dotu {
            fc.Errornum = buf.Read32()
        } else {
            fc.Errornum = 0
        }

    case Twalk:
        fc.Fid = buf.Read32()
        fc.Newfid = buf.Read32()
        m := buf.Read16()
        fc.Wname = make([]string, m)
        for i := 0; i < int(m); i++ {
            fc.Wname[i] = buf.ReadString()
        }

    case Rwalk:
        count := buf.Read16()
        fc.Wqid = make([]Qid, count)
        for i := 0; i < int(count); i++ {
            gqid(buf, &fc.Wqid[i])
        }

    case Topen:
        fc.Fid = buf.Read32()
        fc.Mode = buf.Read8()

    case Ropen, Rcreate:
        gqid(buf, &fc.Qid)
        fc.Iounit = buf.Read32()
        fc.Fid = buf.Read32()
        fc.Mode = buf.Read8()

    case Tcreate:
        fc.Fid = buf.Read32()
        fc.Name = buf.ReadString()
        fc.Perm = buf.Read32()
        fc.Mode = buf.Read8()
        if dotu {
            fc.Ext = buf.ReadString()
        }

    case Tread:
        fc.Fid = buf.Read32()
        fc.Offset = buf.Read64()
        fc.Count = buf.Read32()

    case Rread:
        fc.Count = buf.Read32()
        buf.ReadBytes(int(fc.Count))

    case Twrite:
        fc.Fid = buf.Read32()
        fc.Offset = buf.Read64()
        fc.Count = buf.Read32()

    case Rwrite:
        fc.Count = buf.Read32()

    case Tclunk, Tremove, Tstat:
        fc.Fid = buf.Read32()

    case Rstat:
        buf.Read16() // Eat size.
        gstat(buf, &fc.Dir, dotu)

    case Twstat:
        fc.Fid = buf.Read32()
        buf.Read16() // Eat size.
        gstat(buf, &fc.Dir, dotu)

    case Rflush, Rclunk, Rremove, Rwstat:
        break

    default:
        return nil, InvalidMessage
    }

    if buf.ReadLeft() < 0 {
        log.Printf("buffer overrun? -> %s", fc.String())
    }

    return fc, err
}
