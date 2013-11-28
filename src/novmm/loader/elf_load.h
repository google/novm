/*
 * elf_load.h
 */

#include <sys/types.h>

extern int doLoad(void *self, size_t offset, void *source, size_t length);

long long elf_load(void *elf_start, void *self, int *is_64bit);
