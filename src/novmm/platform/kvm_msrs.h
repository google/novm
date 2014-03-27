/*
 * msrs.h
 */

/* Return the value needed for a single MSR entry. */
int msr_size(void);

/* Set the index and value for a single MSR entry. */
void msr_set(void *data, __u32 index, __u64 value);

/* Extract the value from a single MSR entry. */
__u64 msr_get(void *data);

/* Extract an index from an MSR list. */
int msr_list_index(void *data, int n, __u32 *index);
