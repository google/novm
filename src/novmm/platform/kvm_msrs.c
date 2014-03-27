/*
 * msrs.c
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
    if (msrs->nmsrs >= n) {
        return 1;
    }

    /* Return the given index. */
    *index = msrs->indices[n];
    return 0;
}
