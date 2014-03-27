// +build linux
package platform

/*
#include <linux/kvm.h>
#include "cpuid.h"

// IOCTL calls.
const int CreateVcpu = KVM_CREATE_VCPU;
const int SetGuestDebug = KVM_SET_GUEST_DEBUG;
const int SetMpState = KVM_SET_MP_STATE;

// States.
const int MpStateRunnable = KVM_MP_STATE_RUNNABLE;
const int MpStateUninitialized = KVM_MP_STATE_UNINITIALIZED;
const int MpStateInitReceived = KVM_MP_STATE_INIT_RECEIVED;
const int MpStateHalted = KVM_MP_STATE_HALTED;
const int MpStateSipiReceived = KVM_MP_STATE_SIPI_RECEIVED;

// IOCTL flags.
const int GuestDebugEnable = KVM_GUESTDBG_ENABLE|KVM_GUESTDBG_SINGLESTEP;
*/
import "C"

import (
    "log"
    "sync/atomic"
    "syscall"
    "unsafe"
)

type Vcpu struct {
    // The VCPU fd.
    fd  int

    // The mmap-structure.
    // NOTE: mmap is the go pointer to the bytes,
    // kvm points to same data but is interpreted.
    mmap []byte
    kvm  *C.struct_kvm_run

    // Cached registers.
    // See data.go for the serialization code.
    regs  C.struct_kvm_regs
    sregs C.struct_kvm_sregs

    // Caching parameters.
    regs_cached  bool
    sregs_cached bool
    regs_dirty   bool
    sregs_dirty  bool

    // Our available MSRs.
    msrs []uint32

    // Our default cpuid.
    cpuid []Cpuid

    // Is this stepping?
    is_stepping bool

    // Our run information.
    RunInfo
}

func (vm *Vm) NewVcpu() (*Vcpu, error) {

    // Create a new Vcpu.
    id := atomic.AddInt32(&vm.next_id, 1) - 1
    log.Printf("kvm: creating VCPU (id: %d)...", id)
    vcpufd, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.CreateVcpu),
        uintptr(id))
    if e != 0 {
        return nil, e
    }

    // Make sure this disappears on exec.
    // This is a race condition here, but realistically
    // we are not going to be restarting at this point.
    syscall.CloseOnExec(int(vcpufd))

    // Map our shared data.
    log.Printf("kvm: mapping VCPU shared state...")
    mmap, err := syscall.Mmap(
        int(vcpufd),
        0,
        vm.mmap_size,
        syscall.PROT_READ|syscall.PROT_WRITE,
        syscall.MAP_SHARED)
    if err != nil {
        syscall.Close(int(vcpufd))
        return nil, err
    }
    kvm_run := (*C.struct_kvm_run)(unsafe.Pointer(&mmap[0]))

    // Add our Vcpu.
    vcpu := &Vcpu{
        fd:    int(vcpufd),
        mmap:  mmap,
        kvm:   kvm_run,
        msrs:  vm.msrs,
        cpuid: vm.cpuid,
    }
    vm.vcpus = append(vm.vcpus, vcpu)

    // Set our default cpuid.
    // (This may be overriden later).
    err = vcpu.SetCpuid(vm.cpuid)
    if err != nil {
        vcpu.Dispose()
        return nil, err
    }

    // Return our VCPU object.
    return vcpu, vcpu.initRunInfo()
}

func (vcpu *Vcpu) Dispose() error {

    // Halt the processor.
    var mp_state C.struct_kvm_mp_state
    mp_state.mp_state = C.__u32(C.MpStateHalted)
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vcpu.fd),
        uintptr(C.SetMpState),
        uintptr(unsafe.Pointer(&mp_state)))
    if e != 0 {
        return e
    }

    // Cleanup our resources.
    syscall.Munmap(vcpu.mmap)
    return syscall.Close(vcpu.fd)
}

func (vcpu *Vcpu) SetStepping(step bool) error {

    var guest_debug C.struct_kvm_guest_debug

    if step == vcpu.is_stepping {
        // Already set.
        return nil
    } else if step {
        guest_debug.control = C.__u32(C.GuestDebugEnable)
    } else {
        guest_debug.control = 0
    }

    // Execute our debug ioctl.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vcpu.fd),
        uintptr(C.SetGuestDebug),
        uintptr(unsafe.Pointer(&guest_debug)))
    if e != 0 {
        return e
    }

    // We're okay.
    vcpu.is_stepping = step
    return nil
}

func (vcpu *Vcpu) IsStepping() bool {
    return vcpu.is_stepping
}
