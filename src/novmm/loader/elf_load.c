/*
 * elf_load.c
 *
 * Implementation of a simple linux loader.
 *
 * Note that this is only part of the implementation,
 * I've managed to keep most of the bit twiddling in
 * go itself -- this is simply easier because of the
 * direct access to the ELF type definitions.
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

#include <elf.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>

#define ELF_LOAD(phdr, phnum, elf_start, self)   \
do {                                             \
    int i;                                       \
    for(i=0; i < phnum; i++) {                   \
        int r;                                   \
        if(phdr[i].p_type != PT_LOAD)            \
            continue;                            \
        if(phdr[i].p_filesz > phdr[i].p_memsz)   \
            return -EINVAL;                      \
        if(!phdr[i].p_filesz)                    \
            return -EINVAL;                      \
        r = doLoad(                              \
            self,                                \
            phdr[i].p_paddr,                     \
            (char*)elf_start + phdr[i].p_offset, \
            phdr[i].p_filesz);                   \
        if(r != 0)                               \
            return r;                            \
    }                                            \
    return 0;                                    \
} while(0)

int
elf32_load(
    Elf32_Phdr *phdr,
    int phnum,
    void *elf_start,
    void *self)
{
    ELF_LOAD(phdr, phnum, elf_start, self);
}

int
elf64_load(
    Elf64_Phdr *phdr,
    int phnum,
    void *elf_start,
    void *self)
{
    ELF_LOAD(phdr, phnum, elf_start, self);
}

long long
elf_load(
    char *elf_start,
    void *self,
    int *is_64bit)
{
    Elf32_Ehdr *hdr32 = (Elf32_Ehdr*)elf_start;
    Elf64_Ehdr *hdr64 = (Elf64_Ehdr*)elf_start;

    if (hdr32->e_ident[EI_MAG0] != ELFMAG0 ||
        hdr32->e_ident[EI_MAG1] != ELFMAG1 ||
        hdr32->e_ident[EI_MAG2] != ELFMAG2 ||
        hdr32->e_ident[EI_MAG3] != ELFMAG3)
        return -EINVAL;

    if (hdr32->e_ident[4] == ELFCLASS32) {
        int r = elf32_load(
            (Elf32_Phdr *)(elf_start + hdr32->e_phoff),
            hdr32->e_phnum,
            elf_start,
            self);
        if (r<0)
            return r;
        *is_64bit = 0;
        return hdr32->e_entry;

    } else if (hdr64->e_ident[4] == ELFCLASS64) {
        int r = elf64_load(
            (Elf64_Phdr *)(elf_start + hdr64->e_phoff),
            hdr64->e_phnum,
            elf_start,
            self);
        if (r<0)
            return r;
        *is_64bit = 1;
        return hdr64->e_entry;

    } else {
        return -EINVAL;
    }
}
