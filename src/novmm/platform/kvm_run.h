/*
 * kvm_run.h
 *
 * C-stub to enter into guest mode.
 *
 * This exists because of complexities around go-routine
 * scheduling and the ability to deliver a signal to a specific
 * thread. Because we are not able to control this from Go,
 * we need to isolate some calls in C for pause/unpause control.
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
