/*
 * cpuid.c
 */

#include <errno.h>
#include <linux/kvm.h>
#include "kvm_cpuid.h"

void cpuid_init(void *data, int size) {
    struct kvm_cpuid2 *cpuid = (struct kvm_cpuid2*)data;
    cpuid->nent = (size - sizeof(struct kvm_cpuid2)) / sizeof(struct kvm_cpuid_entry);
}

int cpuid_get(void *data, int n, __u32 *function, __u32 *eax, __u32 *ebx, __u32 *ecx, __u32 *edx) {
    struct kvm_cpuid2 *cpuid = (struct kvm_cpuid2*)data;

    /* Off the end? */
    if (n >= cpuid->nent) {
        return E2BIG;
    }

    /* Extract from the structure. */
    *function = cpuid->entries[n].function;
    *eax = cpuid->entries[n].eax;
    *ebx = cpuid->entries[n].ebx;
    *ecx = cpuid->entries[n].ecx;
    *edx = cpuid->entries[n].edx;
    return 0;
}

void cpuid_native(__u32 function, __u32 *eax, __u32 *ebx, __u32 *ecx, __u32 *edx) {
    /* Get our native cpuid. */
    asm volatile("cpuid"
        :"=a"(*eax),"=b"(*ebx),"=c"(*ecx),"=d"(*edx)
        :"a"(function));
}

int cpuid_set(void *data, int size, int n, __u32 function, __u32 eax, __u32 ebx, __u32 ecx, __u32 edx) {
    struct kvm_cpuid2 *cpuid = (struct kvm_cpuid2*)data;

    /* Is it too big? */
    if ((sizeof(struct kvm_cpuid2) + n*sizeof(struct kvm_cpuid_entry)) > size) {
        return ENOMEM;
    }

    /* Set the entry as specified. */
    cpuid->entries[n].function = function;
    cpuid->entries[n].eax = eax;
    cpuid->entries[n].eax = ebx;
    cpuid->entries[n].eax = ecx;
    cpuid->entries[n].eax = edx;
    cpuid->nent = n;
    return 0;
}
