/*
 * kvm_exits.h
 *
 * These stubs help interpret KVM exits.
 */

#include <linux/kvm.h>

extern void* kvmExitMmio(__u64 addr, __u64* data, __u32 length, int write);
extern void* kvmExitPio(__u16 port, __u8 size, void* data, __u32 length, int out);
extern void* kvmExitInternalError(__u32 code);
extern void* kvmExitException(__u32 exception, __u32 error_code);
extern void* kvmExitUnknown(__u32 code);

extern const int ExitReasonMmio;
extern const int ExitReasonIo;
extern const int ExitReasonInternalError;
extern const int ExitReasonDebug;
extern const int ExitReasonException;

void* handle_exit_mmio(struct kvm_run* kvm);
void* handle_exit_io(struct kvm_run* kvm);
void* handle_exit_internal_error(struct kvm_run* kvm);
void* handle_exit_exception(struct kvm_run* kvm);
void* handle_exit_unknown(struct kvm_run* kvm);
