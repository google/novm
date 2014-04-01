/*
 * virtio.h
 */

#include <linux/virtio_ring.h>

int vring_get_buf(
    struct vring* vring,
    __u16 consumed,
    __u16* flags,
    __u16* index,
    __u16* used_event);

void vring_read_desc(
    struct vring_desc* desc,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next);

void vring_get_index(
    struct vring* vring,
    __u16 index,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next);

void vring_put_buf(
    struct vring* vring,
    __u16 index,
    __u32 len,
    int* evt_interrupt,
    int* no_interrupt);
