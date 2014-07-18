/*
 * kvm_run.c
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

#include <signal.h>
#include <errno.h>
#include <stdlib.h>
#include <pthread.h>
#include <string.h>
#include <sys/ioctl.h>
#include <linux/kvm.h>

#include "kvm_run.h"

int kvm_run_init(int vcpufd, struct kvm_run_info *info) {
    int rval = 0;
    sigset_t set;
    struct kvm_signal_mask *arg;

    arg = malloc(sizeof(*arg) + sizeof(set));
    if (arg == NULL) {
        return ENOMEM;
    }

    /* Initialize our lock. */
    if (pthread_mutex_init(&info->lock, NULL) < 0) {
        return errno;
    }

    /* Enable all signals. */
    sigemptyset(&set);
    arg->len = 8;
    memcpy(arg->sigset, &set, sizeof(set));

    /* Set the mask during KVM_RUN. */
    rval = ioctl(vcpufd, KVM_SET_SIGNAL_MASK, arg);
    free(arg);

    return rval < 0 ? errno : 0;
}

int kvm_run(int vcpufd, int sig, struct kvm_run_info *info) {
    int rval = 0;
    sigset_t newset;
    sigset_t oldset;

    /* Acquire our lock. */
    pthread_mutex_lock(&info->lock);

    /* Did we receive a cancel request? */
    if (info->cancel) {
        info->cancel = 0;
        pthread_mutex_unlock(&info->lock);
        return EINTR;
    }

    /* Block our interrupt signal temporarily. */
    sigemptyset(&newset);
    sigaddset(&newset, sig);
    if (pthread_sigmask(SIG_BLOCK, &newset, &oldset) < 0) {
        pthread_mutex_unlock(&info->lock);
        return errno;
    }

    /* Save our tid. */
    info->tid = pthread_self();
    info->running = 1;

    /* Drop our lock, we're now "running".
     * After the signal was blocked above, we
     * were guaranteed that anyone who acquires
     * the lock, reads the TID and does a kill
     * will actually interrupt the KVM_RUN. */
    pthread_mutex_unlock(&info->lock);

    /* Enter into guest mode. */
    rval = ioctl(vcpufd, KVM_RUN, 0);
    if (rval < 0) {
        rval = errno;
    }

    /* Reacquire our lock. */
    pthread_mutex_lock(&info->lock);

    /* Note that we are no longer running.
     * It's quite possible that prior to acquiring
     * the lock above, someone ma hit us with another
     * signal. This is okay, it will be consumed after
     * we unblock the signal block (harmlessly). */
    info->running = 0;
    info->cancel = 0;

    /* Done with clearing running & cancel. */
    pthread_mutex_unlock(&info->lock);

    /* Unblock the interrupt signal signal. */
    if (pthread_sigmask(SIG_SETMASK, &oldset, NULL) < 0) {
        return rval != 0 ? rval : errno;
    }

    return rval;
}

int kvm_run_interrupt(int vcpufd, int sig, struct kvm_run_info *info) {
    (void)vcpufd;

    /* Acquire our lock. */
    pthread_mutex_lock(&info->lock);

    /* Is this thread running? */
    if (info->running) {
        pthread_kill(info->tid, sig);
    } else {
        info->cancel = 1;
    }

    /* We're done. */
    pthread_mutex_unlock(&info->lock);
}
