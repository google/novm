/*
 * virtio.c
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

#include "virtio.h"

int vring_get_buf(
    struct vring* vring,
    __u16 consumed,
    __u16* flags,
    __u16* index,
    __u16* used_event) {

    if (consumed != vring->avail->idx) {

        if (consumed+1 < vring->avail->idx) {
            vring->used->flags = VRING_USED_F_NO_NOTIFY;
        } else {
            vring->used->flags = 0;
            vring_avail_event(vring) = consumed+1;
        }

        *flags = vring->avail->flags;
        *index = vring->avail->ring[consumed%vring->num];

        return 1;
    }

    return 0;
}

void vring_read_desc(
    struct vring_desc* desc,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next) {

    *addr = desc->addr;
    *len = desc->len;
    *flags = desc->flags;
    *next = desc->next;
}

void vring_get_index(
    struct vring* vring,
    __u16 index,
    __u64* addr,
    __u32* len,
    __u16* flags,
    __u16* next) {

    vring_read_desc(&vring->desc[index], addr, len, flags, next);
}

void vring_put_buf(
    struct vring* vring,
    __u16 index,
    __u32 len,
    int* evt_interrupt,
    int* no_interrupt) {

    vring->used->ring[vring->used->idx%vring->num].id = index;
    vring->used->ring[vring->used->idx%vring->num].len = len;
    *evt_interrupt = vring_used_event(vring) == vring->used->idx;
    *no_interrupt = vring->used->flags & VRING_AVAIL_F_NO_INTERRUPT;

    asm volatile ("" : : : "memory");
    vring->used->idx += 1;
    asm volatile ("" : : : "memory");
}
