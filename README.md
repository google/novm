novm
====

*novm* is a legacy-free, type 2 hypervisor written in Go. Its goal is to
provide an alternate, high-performance Linux hypervisor for cloud workloads.

*novm* is powerful because it exposes a filesystem-device as a primary
mechanism for running guests. This allows you to easily manage independent
software and data bundles independently and combine them into a single virtual
machine instance.

*novm* leverages the excellent Linux Kernel Virtual Machine (KVM) interface to
run guest instances.

Why *novm*?
-----------

*novm* changes the rules of virtualization by principally exposing a flexible
filesystem interface instead of virtual block devices. This eliminates the pain
of managing virtual disk images and allows much greater flexibility to how
software is bundled and deployed. A virtual machine is no longer a heavyweight
instance, but rather a hardware-enforced container around a collection of files
and services defined on-the-fly.

*novm* also focuses only on high-performance paravirtualized devices, and drops
support for legacy hardware and most emulation. This makes it inappropriate
for running legacy applications, but ideal for modern cloud-based virtualization
use cases. *novm* was originally called *pervirt*, but this name was changed
after it was suggested that this name *could* be misconstrued.

Technical Details
-----------------

### Bootloader ###

Currently, the in-built bootloader supports only modern Linux kernels.

It follows the Linux boot convention by setting the vcpu directly into 32-bit
protected mode and jumping to the ELF entry point. Or, if it finds a 64-bit
kernel, it will do a bit of additional setup (creating an identity-mapped page
table) and then jumps directly into 64-bit long mode at the entry point.

For full details on this convention see:
    https://www.kernel.org/doc/Documentation/x86/boot.txt

#### How does it work? ####

The embedded bootloader works different than traditional bootloaders, such as
GRUB or LILO. Unlike other bootloaders, the embedded bootloader requires the
ELF kernel binary (vmlinux), not the compressed image (bzImage).

This is because the compressed image (bzImage) contains a compressed version of
the ELF kernel binary, real-mode setup code, and a small setup sector.
Traditional bootloaders typically lay these components out in memory and
execute the real-mode code. The real-mode setup code will construct a few basic
data structures via BIOS calls, extract the vmlinux binary into memory and
finally jump into 32-bit protected mode in the newly uncompressed code.

This is a sensible approach, as the real-mode kernel code will be able to
execute arbitrary BIOS calls and probe the hardware in arbitrary ways before
finally switching to protected mode. However, for a virtualized environment
where the hardware is fixed and known to the hypervisor, the bootloader itself
can lay out the necessary data structures in memory and start *directly* in
32-bit protected mode. In addition to skipping a batch of real-mode execution
and emulation, this allows us to avoid having to build a BIOS at all.

If a bzImage is provided, the ELF binary will be extracted and cached using a
simple script derived from the script found in the Linux tree.

Given a vmlinux binary file, we load the file directly into memory as specified
by the ELF program headers. For example:

    Type           Offset             VirtAddr           PhysAddr
                   FileSiz            MemSiz              Flags  Align
    LOAD           0x0000000000200000 0xffffffff81000000 0x0000000001000000
                   0x0000000000585000 0x0000000000585000  R E    200000
    LOAD           0x0000000000800000 0xffffffff81600000 0x0000000001600000
                   0x000000000009d0f0 0x000000000009d0f0  RW     200000
    LOAD           0x0000000000a00000 0x0000000000000000 0x000000000169e000
                   0x0000000000014bc0 0x0000000000014bc0  RW     200000
    LOAD           0x0000000000ab3000 0xffffffff816b3000 0x00000000016b3000
                   0x00000000000d5000 0x00000000005dd000  RWE    200000
    NOTE           0x000000000058efd0 0xffffffff8138efd0 0x000000000138efd0
                   0x000000000000017c 0x000000000000017c         4

We also note the entry point for the binary (here it happens to be 0x1000000).

Before we able to jump to the entry point, we need to setup some basic
requirements that would normally be done by the real-mode setup code:

* Need a simple GDT and BOOT_CS and BOOT_DS.
* An empty IDT is fine.
* Interrupts need to be disabled.
* CR0 is set appropriately to enable protected mode.
* Registers are cleared appropriately (per boot convention).
* Boot params structure is initialized and pointed to (see below).

As part of this setup process, any provided initial ram disk is also loaded
into memory. This is pointed to as part of the boot parameters, and
decompression is handled by the kernel.

### Devices ###

The device model implemented by this kernel is substantially different from
most VMs. Instead of attemping to emulate a legacy machine, we instead to
support only the mechanisms which make sense for a next generation
purpose-built Virtual Machine.

#### What does it not have? ####

* No I/O ports.

There are no supported I/O port devices.

* No PCI bus.

Nope, none of that either.

* No ACPI.

Instead, we use a device tree to provide a full hardware specification to the
guest operating system. This eliminates waste and layers of abstraction by
allowing the hypervisor to very cleanly and completely specify available
devices.

#### What does it have? ####

* Memory-mapped Virt-I/O devices in user-space.

We only support efficient paravirtualized devices via a direct map.

* In-kernel LAPICs, IOAPIC, and PIT.

Basic emulation is provided in-kernel, eliminating transitions.

* Eventfd-driven interrupts.

The only exits to user-space are those to configure the virtual device.
