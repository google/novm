package machine

/*
#include <stdio.h>
#include <errno.h>
#include <sys/uio.h>

static inline int do_iovec(
    int fd,
    int count,
    void** ptrs,
    int* sizes,
    off_t offset,
    int write) {

    int vecno;
    struct iovec vec[count];
    size_t rval = 0;

    for (vecno = 0; vecno < count; vecno += 1) {
        vec[vecno].iov_base = (char*)ptrs[vecno];
        vec[vecno].iov_len = sizes[vecno];
    }

    if (offset != (off_t)-1) {
        if (write) {
            rval = pwritev(fd, &vec[0], count, offset);
        } else {
            rval = preadv(fd, &vec[0], count, offset);
        }
    } else {
        if (write) {
            rval = writev(fd, &vec[0], count);
        } else {
            rval = readv(fd, &vec[0], count);
        }
    }

    if (rval < 0) {
        return -errno;
    }

    return rval;
}
*/
import "C"

import (
    "syscall"
    "unsafe"
)

//
// The virtIO buffer is a collection of different
// descriptor objects. It exposes simple header
// manipulation primitives as well as scatter-gather
// I/O operations for zero-copy efficiency.
//

type VirtioBuffer struct {
    data     [][]byte
    index    uint16
    length   int
    readonly bool
}

func NewVirtioBuffer(index uint16, readonly bool) *VirtioBuffer {
    buf := new(VirtioBuffer)
    buf.data = make([][]byte, 0, 1)
    buf.index = index
    buf.readonly = readonly
    buf.length = 0
    return buf
}

func (buf *VirtioBuffer) Append(data []byte) {
    buf.data = append(buf.data, data)
    buf.length += len(data)
}

func (buf *VirtioBuffer) Length() int {
    return buf.length
}

func (buf *VirtioBuffer) SetLength(length int) {
    buf.length = length
}

func (buf *VirtioBuffer) Gather(
    offset int,
    length int) ([]unsafe.Pointer, []C.int) {

    ptrs := make([]unsafe.Pointer, 0, len(buf.data))
    lens := make([]C.int, 0, len(buf.data))

    for _, data := range buf.data {
        if offset >= len(data) {
            offset -= len(data)
        } else if offset > 0 {
            ptrs = append(ptrs, unsafe.Pointer(&data[offset]))
            if len(data)-offset > length {
                lens = append(lens, C.int(length))
                length = 0
            } else {
                lens = append(lens, C.int(len(data)-offset))
                length -= len(data) - offset
            }
            offset = 0
        } else {
            ptrs = append(ptrs, unsafe.Pointer(&data[0]))
            if len(data) > length {
                lens = append(lens, C.int(length))
                length = 0
            } else {
                lens = append(lens, C.int(len(data)))
                length -= len(data)
            }
        }

        if length == 0 {
            break
        }
    }

    return ptrs, lens
}

func (buf *VirtioBuffer) doIO(
    fd int,
    fd_offset int64,
    buf_offset int,
    length int,
    write C.int) (int, error) {

    // Gather the appropriate elements.
    ptrs, lens := buf.Gather(buf_offset, length)

    // Actually execute our readv/writev.
    rval := C.do_iovec(
        C.int(fd),
        C.int(len(ptrs)),
        &ptrs[0],
        &lens[0],
        C.off_t(fd_offset),
        write)
    if rval < 0 {
        return 0, syscall.Errno(int(-rval))
    }

    return int(rval), nil
}

func (buf *VirtioBuffer) Write(
    fd int,
    buf_offset int,
    length int) (int, error) {

    return buf.doIO(fd, -1, buf_offset, length, C.int(1))
}

func (buf *VirtioBuffer) PWrite(
    fd int,
    fd_offset int64,
    buf_offset int,
    length int) (int, error) {

    return buf.doIO(fd, fd_offset, buf_offset, length, C.int(1))
}

func (buf *VirtioBuffer) Read(fd int,
    buf_offset int,
    length int) (int, error) {

    return buf.doIO(fd, -1, buf_offset, length, C.int(0))
}

func (buf *VirtioBuffer) PRead(
    fd int,
    fd_offset int64,
    buf_offset int,
    length int) (int, error) {

    return buf.doIO(fd, fd_offset, buf_offset, length, C.int(0))
}

func (buf *VirtioBuffer) Map(
    offset int,
    length int) Ram {

    for _, data := range buf.data {
        if offset >= len(data) {
            offset -= len(data)
        } else if offset > 0 {
            if length > len(data)-offset {
                return Ram(data[offset:len(data)])
            } else {
                return Ram(data[offset : offset+length])
            }
        } else {
            if length > len(data) {
                return Ram(data)
            } else {
                return Ram(data[:length])
            }
        }
    }

    // We never found the offset,
    // give back an empty array.
    return []byte{}
}
