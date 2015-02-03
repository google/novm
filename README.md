*This is not an official Google product.*

novm
====

*novm* is a legacy-free, type 2 hypervisor written in Go. Its goal is to
provide an alternate, high-performance Linux hypervisor for cloud workloads.

*novm* is unique because it exposes a filesystem-device as a primary
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

*novm* aims to provide the best of both containers and hardware virtualization.

What are the advantages over containers?
----------------------------------------

* You can run any compatible kernel.

You have much more freedom as a result of the fact that the
guest kernel is decoupled from the host.

For example: You can hold back your guest kernel if necessary,
or upgrade to the latest and greatest. Combined with migration,
you can have a stable, long-running guest while still applying
critical fixes and updates to the hosts. If you need filesystem
or networking modules, you can freely load them in a guest. If
a particular application requires a specific kernel, it's very
straight-forward to use it.

* You have real, hardware-enforced isolation.

The only interface exposed is the x86 ABI and the hypervisor.
Containers are more likely to suffer from security holes as the
guest can access the entire kernel system call interface.

In multi-tenant environments, strong isolation is very important.
Beyond security holes, containers likely present countless
opportunities for subtle information leaks between guest instances.

* You can mix and match technologies.

Want a docker-style *novm*? Want a disk-based *novm*? Sure.
Both co-exist easily and use resources in the same way. There's
no need to manage two separate networking systems, bridges,
management daemons, etc. Everything is managed in one way.

What are the disadvantages over containers?
-------------------------------------------

* Performance.

Okay, so there's a non-trivial hit. If your workload is very I/O
intensive (massive disk or network usage), you will see a small
but measurable drop in performance. But most workloads will not
see a difference.

(Note that this project is still experimental, and there's plenty
of performance work still to be done. The above should be considered
a forward-looking statement.)

What are the advantages over traditional virtualization?
--------------------------------------------------------

* File-based provisioning.

Instead of opaque disk images, *novm* allows you to easily
combine directory trees to create a VM in real-time. This allows
an administrator to very easily tweak files and fire up VMs
without dealing with the headaches of converting disk images or
using proprietary tools.

What are the disadvantages over traditional virtualization?
-----------------------------------------------------------

* Legacy hardware support.

We only support a very limited number of devices. This means
that guests must be VirtIO-aware and not depend on the presence
of any BIOS functionality post-boot.

For new workloads, this isn't a problem. But it means you can't
migrate your untouchable, ancient IT system over to *novm*.

* Arbitrary guest OS support.

*novm* only supports the guests we know how to boot. For
the time being, that means Linux only. It would be straight
forward to add support for multiboot guests, like FreeBSD.

Requirements
------------

*novm* requires at least go 1.1 in order to build.

*novm* also requires at least Linux 3.8, with appropriate headers. If your
headers are not recent enough, you may see compilation errors related to
`KVM_SIGNAL_MSI` or `KVM_CAP_SIGNAL_MSI`.

You also need Python 2.6+ or Python 3+, plus the Python six module.

Building
---------

To build *novm*, simply clone the repo and run `make`.

Optionally, you can build packages by running `make deb` or `make rpm`.

To run *novm*, use `scripts/novm`, or if you've installed packages, just type `novm`.

For information on using the command line, just type `novm`. You may use
`novm <command> --help` for detailed information on any specific command.

The first thing you'll need is a kernel. You can create a kernel bundle using
the running kernel by running `novm-import-kernel`.

To create a novm instance, try using `novm create --com1 /bin/ls`.

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

Before we are able to jump to the entry point, we need to setup some basic
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
