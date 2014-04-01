/*
 * virtio_buffer.c
 */

#include <errno.h>
#include <sys/uio.h>
#include "virtio_buffer.h"

int do_iovec(
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
