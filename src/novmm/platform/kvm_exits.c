/*
 * kvm_exits.c
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

#include "kvm_exits.h"

/* Exit reasons. */
const int ExitReasonMmio = KVM_EXIT_MMIO;
const int ExitReasonIo = KVM_EXIT_IO;
const int ExitReasonInternalError = KVM_EXIT_INTERNAL_ERROR;
const int ExitReasonException = KVM_EXIT_EXCEPTION;
const int ExitReasonDebug = KVM_EXIT_DEBUG;
const int ExitReasonShutdown = KVM_EXIT_SHUTDOWN;

void* handle_exit_mmio(struct kvm_run* kvm) {
    return kvmExitMmio(
        kvm->mmio.phys_addr,
        ((__u64*)&(kvm->mmio.data[0])),
        kvm->mmio.len,
        kvm->mmio.is_write);
}

void* handle_exit_io(struct kvm_run* kvm) {
    return kvmExitPio(
        kvm->io.port,
        kvm->io.size,
        (void*)((unsigned long long)kvm + (unsigned long long)kvm->io.data_offset),
        kvm->io.count,
        kvm->io.direction == KVM_EXIT_IO_OUT);
}

void* handle_exit_internal_error(struct kvm_run* kvm) {
    return kvmExitInternalError(kvm->internal.suberror);
}

void* handle_exit_exception(struct kvm_run* kvm) {
    return kvmExitException(kvm->ex.exception, kvm->ex.error_code);
}

void* handle_exit_unknown(struct kvm_run* kvm) {
    return kvmExitUnknown(kvm->exit_reason);
}
