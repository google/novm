/*
 * virtio_buffer.c
 *
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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
