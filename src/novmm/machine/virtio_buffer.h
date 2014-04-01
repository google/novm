/*
 * virtio_buffer.h
 */

#include <stdio.h>

int do_iovec(
    int fd,
    int count,
    void** ptrs,
    int* sizes,
    off_t offset,
    int write);
