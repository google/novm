/*
 * cpuid.h
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
