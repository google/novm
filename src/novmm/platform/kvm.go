// +build linux
package platform

/*
#include <linux/kvm.h>

// IOCTL calls.
const int GetApiVersion = KVM_GET_API_VERSION;
const int CreateVm = KVM_CREATE_VM;
const int CreateVcpu = KVM_CREATE_VCPU;
const int GetVcpuMmapSize = KVM_GET_VCPU_MMAP_SIZE;
const int CheckExtension = KVM_CHECK_EXTENSION;
const int SetUserMemoryRegion = KVM_SET_USER_MEMORY_REGION;
const int CreateIrqChip = KVM_CREATE_IRQCHIP;
const int IrqLine = KVM_IRQ_LINE;
const int CreatePit2 = KVM_CREATE_PIT2;
const int SetGuestDebug = KVM_SET_GUEST_DEBUG;
const int SetMpState = KVM_SET_MP_STATE;
const int Translate = KVM_TRANSLATE;
const int GetSupportedCpuid = KVM_GET_SUPPORTED_CPUID;
const int SetCpuid = KVM_SET_CPUID2;

// States.
const int MpStateRunnable = KVM_MP_STATE_RUNNABLE;
const int MpStateUninitialized = KVM_MP_STATE_UNINITIALIZED;
const int MpStateInitReceived = KVM_MP_STATE_INIT_RECEIVED;
const int MpStateHalted = KVM_MP_STATE_HALTED;
const int MpStateSipiReceived = KVM_MP_STATE_SIPI_RECEIVED;

// IOCTL flags.
const int MemLogDirtyPages = KVM_MEM_LOG_DIRTY_PAGES;
const int GuestDebugFlags = KVM_GUESTDBG_ENABLE|KVM_GUESTDBG_SINGLESTEP;

// Capabilities (extensions).
const int CapUserMem = KVM_CAP_USER_MEMORY;
const int CapIrqChip = KVM_CAP_IRQCHIP;
const int CapIoFd = KVM_CAP_IOEVENTFD;
const int CapIrqFd = KVM_CAP_IRQFD;
const int CapPit2 = KVM_CAP_PIT2;
const int CapGuestDebug = KVM_CAP_SET_GUEST_DEBUG;
const int CapCpuid = KVM_CAP_EXT_CPUID;

// We need to fudge the types for irq level.
// This is because of the extremely annoying semantics
// for accessing *unions* in Go. Basically it can't.
// See the description below in createIrqChip().
struct irq_level {
    __u32 irq;
    __u32 level;
};
static int check_irq_level(void) {
    if (sizeof(struct kvm_irq_level) != sizeof(struct irq_level)) {
        return 1;
    } else {
        return 0;
    }
}

static void cpuid_init(void *data, int size) {
    struct kvm_cpuid2 *cpuid = (struct kvm_cpuid2*)data;
    cpuid->nent = (size - sizeof(struct kvm_cpuid2))
        / sizeof(struct kvm_cpuid_entry);
}

static void cpuid_finish(void *data) {
    struct kvm_cpuid2 *cpuid = (struct kvm_cpuid2*)data;
    int n;
    __u32 eax, ebx, ecx, edx;

    for( n = 0; n < cpuid->nent; n += 1 ) {
        if (cpuid->entries[n].function == 0) {
            eax = 0;
            asm volatile("cpuid"
                :"=a"(eax),"=b"(ebx),"=c"(ecx),"=d"(edx)
                :"a"(eax));
            // Copy our vendor.
            cpuid->entries[n].ecx = ecx;
            cpuid->entries[n].ebx = ebx;
            cpuid->entries[n].edx = edx;
        }
        if (cpuid->entries[n].function == 1) {
            eax = 1;
            asm volatile("cpuid"
                :"=a"(eax),"=b"(ebx),"=c"(ecx),"=d"(edx)
                :"a"(eax));
            // Copy our cpu model.
            cpuid->entries[n].eax = eax;
            // Note that we have an APIC.
            cpuid->entries[n].edx |= (1<<9);
        }
    }
}

// NOTE: Not really generally available yet.
// This is a pretty new feature, but once it's available
// it surely will allow rearchitecting some of the MMIO-based
// devices to operate more efficently (as the guest will only
// trap out on WRITEs, and not on READs).
// const int MemReadOnly = KVM_MEM_READONLY;
// const int CapReadOnlyMem = KVM_CAP_READONLY_MEM;
*/
import "C"

import (
    "errors"
    "log"
    "sync"
    "syscall"
    "unsafe"
)

type kvmCapability struct {
    name   string
    number uintptr
}

func (capability *kvmCapability) Error() string {
    return "Missing capability: " + capability.name
}

var requiredCapabilities = []kvmCapability{
    kvmCapability{"User Memory", uintptr(C.CapUserMem)},
    kvmCapability{"IRQ Chip", uintptr(C.CapIrqChip)},
    kvmCapability{"IO Event FD", uintptr(C.CapIoFd)},
    kvmCapability{"IRQ Event FD", uintptr(C.CapIrqFd)},
    kvmCapability{"PIT2", uintptr(C.CapPit2)},
    kvmCapability{"CPUID", uintptr(C.CapCpuid)},

    // It does seem to be the case that this capability
    // is not advertised correctly. On my kernel (3.11),
    // it supports this ioctl but yet claims this capability
    // is not available.
    // In any case, this isn't necessary functionality,
    // but the call to SetSingleStep() may fail.
    // kvmCapability{"Guest debug", uintptr(C.CapGuestDebug)},

    // See NOTE above.
    // kvmCapability{"Read-only Memory", uintptr(C.CapReadOnlyMem)},
}

type Vm struct {
    // The VM fd.
    fd  int

    // The next vcpu to create.
    next_id int

    // The next memory region slot to create.
    // This is not serialized because we will
    // recreate all regions (and the ordering
    // may even be different the 2nd time round).
    mem_region int

    // Our cpuid data.
    // At the moment, we just expose the full
    // host flags to the guest.
    cpuid []byte
}

type Vcpu struct {
    // The VCPU fd.
    fd  int

    // The VCPU id.
    vcpu_id int

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
}

// The size of the mmap structure.
var mmapSize int
var mmapSizeOnce sync.Once
var mmapSizeError error

// Our cpuid data.
var cpuidData []byte
var cpuidDataOnce sync.Once
var cpuidDataError error

func getMmapSize(fd int) {
    // Get the size of the Mmap structure.
    r, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(C.GetVcpuMmapSize),
        0)
    if e != 0 {
        mmapSize = 0
        mmapSizeError = e
    } else {
        mmapSize = int(r)
    }
}

func getCpuidData(fd int) {

    cpuidData = make([]byte, PageSize, PageSize)
    cpuid := unsafe.Pointer(&cpuidData[0])
    C.cpuid_init(cpuid, PageSize)

    for {
        _, _, e := syscall.Syscall(
            syscall.SYS_IOCTL,
            uintptr(fd),
            uintptr(C.GetSupportedCpuid),
            uintptr(unsafe.Pointer(&cpuidData[0])))

        if e == syscall.ENOMEM {
            // The nent field will now have been
            // adjusted, and we can run it again.
            continue
        } else if e != 0 {
            cpuidDataError = e
            break
        }

        // We're good!
        break
    }

    // Finish it off.
    C.cpuid_finish(cpuid)
}

func NewVm() (*Vm, error) {
    fd, err := syscall.Open("/dev/kvm", syscall.O_RDWR, 0)
    if err != nil {
        return nil, err
    }
    defer syscall.Close(fd)

    // Check API version.
    version, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(C.GetApiVersion),
        0)
    if version != 12 || e != 0 {
        return nil, e
    }

    // Check our extensions.
    for _, capSpec := range requiredCapabilities {
        err = checkCapability(fd, capSpec)
        if err != nil {
            return nil, err
        }
    }

    // Make sure we have the mmap size.
    mmapSizeOnce.Do(func() { getMmapSize(fd) })
    if mmapSizeError != nil {
        return nil, mmapSizeError
    }

    // Make sure we have cpuid data.
    cpuidDataOnce.Do(func() { getCpuidData(fd) })
    if cpuidDataError != nil {
        return nil, cpuidDataError
    }

    // Create new VM.
    vmfd, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(C.CreateVm),
        0)
    if e != 0 {
        return nil, e
    }

    // Prepare our VM object.
    log.Print("kvm: VM created.")
    vm := &Vm{fd: int(vmfd)}

    // Try to create an IRQ chip.
    err = vm.createIrqChip()
    if err != nil {
        vm.Dispose()
        return nil, err
    }

    // Create our timer.
    err = vm.createPit()
    if err != nil {
        vm.Dispose()
        return nil, err
    }

    return vm, nil
}

func checkCapability(
    fd int,
    capability kvmCapability) error {

    // Create a new Vcpu.
    // This new Vcpu will already have an in-kernel IRQ chip,
    // as well as an in-kernel PIT chip -- we don't need to worry
    // about emulating them at all. KVM is great.
    r, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(C.CheckExtension),
        capability.number)
    if r != 1 || e != 0 {
        return &capability
    }

    return nil
}

func (vm *Vm) Dispose() error {
    return syscall.Close(vm.fd)
}

func (vm *Vm) VcpuCount() int {
    return vm.next_id + 1
}

func (vm *Vm) NewVcpu() (*Vcpu, error) {
    // Create a new Vcpu.
    vcpu_id := vm.next_id
    log.Printf("kvm: creating VCPU (id: %d)...", vcpu_id)
    vcpufd, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.CreateVcpu),
        uintptr(vcpu_id))
    if e != 0 {
        return nil, e
    }

    // Set our vcpuid.
    _, _, e = syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vcpufd),
        uintptr(C.SetCpuid),
        uintptr(unsafe.Pointer(&cpuidData[0])))
    if e != 0 {
        return nil, e
    }

    // Map our shared data.
    log.Printf("kvm: Mapping VCPU shared state...")
    mmap, err := syscall.Mmap(
        int(vcpufd),
        0,
        mmapSize,
        syscall.PROT_READ|syscall.PROT_WRITE,
        syscall.MAP_SHARED)
    if err != nil {
        syscall.Close(int(vcpufd))
        return nil, err
    }
    kvm_run := (*C.struct_kvm_run)(unsafe.Pointer(&mmap[0]))

    // Bump our next vcpu id.
    vm.next_id += 1

    // Return our VCPU object.
    return &Vcpu{
        fd:      int(vcpufd),
        vcpu_id: vcpu_id,
        mmap:    mmap,
        kvm:     kvm_run}, nil
}

func (vm *Vm) createIrqChip() error {
    // No parameters needed, just create the chip.
    // This is called as the VM is being created in
    // order to ensure that all future vcpus will have
    // their own local apic.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.CreateIrqChip),
        0)
    if e != 0 {
        return e
    }

    // Ugh. A bit of type-fudging. Because of the
    // way go handles unions, we use a custom type
    // for the Interrupt() function below. Let's just
    // check once that everything is sane.
    if C.check_irq_level() != 0 {
        return errors.New("KVM irq_level doesn't match expected!")
    }

    log.Print("kvm: IRQ chip created.")
    return nil
}

func (vm *Vm) createPit() error {
    // Prepare the PIT config.
    // The only flag supported at the time of writing
    // was KVM_PIT_SPEAKER_DUMMY, which I really have no
    // interest in supporting.
    var pit C.struct_kvm_pit_config
    pit.flags = C.__u32(0)

    // Execute the ioctl.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.CreatePit2),
        uintptr(unsafe.Pointer(&pit)))
    if e != 0 {
        return e
    }

    log.Print("kvm: PIT created.")
    return nil
}

func (vm *Vm) Interrupt(
    irq Irq,
    level bool) error {

    // Prepare the IRQ.
    var irq_level C.struct_irq_level
    irq_level.irq = C.__u32(irq)
    if level {
        irq_level.level = C.__u32(1)
    } else {
        irq_level.level = C.__u32(0)
    }

    // Execute the ioctl.
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.IrqLine),
        uintptr(unsafe.Pointer(&irq_level)))
    if e != 0 {
        return e
    }

    return nil
}

func (vm *Vm) MapUserMemory(
    start Paddr,
    size uint64,
    mmap []byte) error {

    // See NOTE above about read-only memory.
    // As we will not support it for the moment,
    // we do not expose it through the interface.
    // Leveraging that feature will likely require
    // a small amount of re-architecting in any case.
    var region C.struct_kvm_userspace_memory_region
    region.slot = C.__u32(vm.mem_region)
    region.flags = C.__u32(0)
    region.guest_phys_addr = C.__u64(start)
    region.memory_size = C.__u64(size)
    region.userspace_addr = C.__u64(uintptr(unsafe.Pointer(&mmap[0])))

    // Execute the ioctl.
    log.Printf(
        "kvm: creating %x byte memory region [%x,%x]...",
        size,
        start,
        uint64(start)+size-1)
    _, _, e := syscall.Syscall(syscall.SYS_IOCTL,
        uintptr(vm.fd),
        uintptr(C.SetUserMemoryRegion),
        uintptr(unsafe.Pointer(&region)))
    if e != 0 {
        return e
    }

    // We're set, bump our slot.
    vm.mem_region += 1
    return nil
}

func (vm *Vm) MapReservedMemory(
    start Paddr,
    size uint64) error {

    // Nothing to do.
    return nil
}

func (vcpu *Vcpu) setSingleStep(on bool) error {

    // Execute our debug ioctl.
    var guest_debug C.struct_kvm_guest_debug
    if on {
        guest_debug.control = C.__u32(C.GuestDebugFlags)
    } else {
        guest_debug.control = 0
    }
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vcpu.fd),
        uintptr(C.SetGuestDebug),
        uintptr(unsafe.Pointer(&guest_debug)))
    if e != 0 {
        return e
    }

    // We're okay.
    return nil
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

func (vcpu *Vcpu) Translate(
    vaddr Vaddr) (Paddr, bool, bool, bool, error) {

    // Perform the translation.
    var translation C.struct_kvm_translation
    translation.linear_address = C.__u64(vaddr)
    _, _, e := syscall.Syscall(
        syscall.SYS_IOCTL,
        uintptr(vcpu.fd),
        uintptr(C.Translate),
        uintptr(unsafe.Pointer(&translation)))
    if e != 0 {
        return Paddr(0), false, false, false, e
    }

    paddr := Paddr(translation.physical_address)
    valid := translation.valid != C.__u8(0)
    writeable := translation.writeable != C.__u8(0)
    usermode := translation.valid != C.__u8(0)

    return paddr, valid, writeable, usermode, nil
}
