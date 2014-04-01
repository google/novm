/*
 * cpuid.h
 */

/* Initialize the structure with nents size appropriately. */
void cpuid_init(void *data, int size);

/* Extract a cpuid value from the structure. */
int cpuid_get(
    void *data,
    int n,
    __u32 *function,
    __u32 *index,
    __u32 *flags,
    __u32 *eax,
    __u32 *ebx,
    __u32 *ecx,
    __u32 *edx);

/* Get a local native cpuid result. */
void cpuid_native(__u32 function, __u32 *eax, __u32 *ebx, __u32 *ecx, __u32 *edx);

/* Set a cpuid value within the structure (updating nents). */
int cpuid_set(
    void *data,
    int size,
    int n,
    __u32 function,
    __u32 index,
    __u32 flags,
    __u32 eax,
    __u32 ebx,
    __u32 ecx,
    __u32 edx);
