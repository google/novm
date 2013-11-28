/*
 * kvm_exits.c
 */

#include "kvm_exits.h"

/* Exit reasons. */
const int ExitReasonMmio = KVM_EXIT_MMIO;
const int ExitReasonIo = KVM_EXIT_IO;
const int ExitReasonInternalError = KVM_EXIT_INTERNAL_ERROR;
const int ExitReasonException = KVM_EXIT_EXCEPTION;
const int ExitReasonDebug = KVM_EXIT_DEBUG;

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
