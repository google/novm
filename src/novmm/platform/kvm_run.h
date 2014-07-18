/*
 * kvm_run.h
 *
 * C-stub to enter into guest mode.
 *
 * This exists because of complexities around go-routine
 * scheduling and the ability to deliver a signal to a specific
 * thread. Because we are not able to control this from Go,
 * we need to isolate some calls in C for pause/unpause control.
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

#include <pthread.h>
#include <linux/kvm.h>

struct kvm_run_info {
    volatile int running;
    volatile int cancel;
    volatile pthread_t tid;

    pthread_mutex_t lock;
};

/* Initialize the signal mask. */
int kvm_run_init(int vcpufd, struct kvm_run_info *info);

/* Prepare for entering guest mode. */
int kvm_run_prep(int vcpufd, struct kvm_run_info *info);

/* Save our tid and enter guest mode. */
int kvm_run(int vcpufd, int sig, struct kvm_run_info *info);

/* Interrupt the running vcpu. */
int kvm_run_interrupt(int vcpufd, int sig, struct kvm_run_info *info);
