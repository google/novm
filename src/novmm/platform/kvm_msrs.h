/*
 * msrs.h
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

/* Return the value needed for a single MSR entry. */
int msr_size(void);

/* Set the index and value for a single MSR entry. */
void msr_set(void *data, __u32 index, __u64 value);

/* Extract the value from a single MSR entry. */
__u64 msr_get(void *data);

/* Extract an index from an MSR list. */
int msr_list_index(void *data, int n, __u32 *index);
