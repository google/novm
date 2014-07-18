/*
 * msrs.c
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

#include <linux/kvm.h>
#include "kvm_msrs.h"

int msr_size(void) {
    return sizeof(struct kvm_msrs) + sizeof(struct kvm_msr_entry);
}

void msr_set(void *data, __u32 index, __u64 value) {
    struct kvm_msrs *msrs = (struct kvm_msrs*)data;

    /* Set as a single value. */
    msrs->nmsrs = 1;
    msrs->entries[0].index = index;
    msrs->entries[0].data = value;
}

__u64 msr_get(void *data) {
    struct kvm_msrs *msrs = (struct kvm_msrs*)data;

    /* Return the (assumed valid) value. */
    return msrs->entries[0].data;
}

int msr_list_index(void *data, int n, __u32 *index) {
    struct kvm_msr_list *msrs = (struct kvm_msr_list*)data;

    /* More than we have? */
    if (n >= msrs->nmsrs) {
        return 1;
    }

    /* Return the given index. */
    *index = msrs->indices[n];
    return 0;
}
