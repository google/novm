"""
Manager entry points.
"""
import os
import sys
import subprocess
import json
import platform
import tempfile
import shutil

from . import utils
from . import db
from . import cli
from . import net
from . import block
from . import serial
from . import basic
from . import memory
from . import fs
from . import clock
from . import pci
from . import cpu
from . import control

class NovmManager(object):

    def __init__(self, root=None):
        if root is None:
            root = os.getenv(
                "NOVM_ROOT",
                os.path.join(
                    os.environ["HOME"],
                    ".novm"))

        self._instances = db.Nodb(
            os.getenv(
                "NOVM_INSTANCES",
                os.path.join(root, "instances")))

        self._packs = db.Nodb(
            os.getenv(
                "NOVM_PACKS",
                os.path.join(root, "packs")))

        self._kernels = db.Nodb(
            os.getenv(
                "NOVM_KERNELS",
                os.path.join(root, "kernels")))

    def run(self,
            name=cli.StrOpt("The instance name."),
            vcpus=cli.IntOpt("The number of vcpus."),
            memsize=cli.IntOpt("The member size (in mb)."),
            kernel=cli.StrOpt("The kernel to use."),
            nic=cli.ListOpt("Define a network device."),
            disk=cli.ListOpt("Define a block device."),
            pack=cli.ListOpt("Use a given read pack."),
            read=cli.ListOpt("Define a backing filesystem read tree."),
            write=cli.ListOpt("Define a backing filesystem write tree."),
            nopci=cli.BoolOpt("Disable PCI devices?"),
            com1=cli.BoolOpt("Enable COM1 UART?"),
            com2=cli.BoolOpt("Enable COM2 UART?"),
            cmdline=cli.StrOpt("Extra command line options?"),
            vmmopt=cli.ListOpt("Options to pass to novmm."),
            nofork=cli.BoolOpt("Don't fork into the background.")):

        """ 
        Run a new instance.

        Note that this does not return (and therefore can
        only be used once inside a 'do' file).

        Network definitions are provided as --nic [opt=val],...

            Available options are:

            name=name             Set the device name.
            debug=true            Enable debugging.
            mac=00:11:22:33:44:55 Set the MAC address.
            tap=tap1              Set the tap name.
            bridge=br0            Enslave to a bridge.
            ip=192.168.1.2/24     Set the IP address.
            gateway=192.168.1.1   Set the gateway IP.

        Disk definitions are provided as --disk [opt=val],...

            Available options are:

            name=name             Set the device name.
            debug=true            Enable debugging.
            file=filename         Set the backing file (raw).
            dev=vda               Set the device name.

        Read definitions are provided as a mapping.

            path=>vm_path         Map the given path for reads.

        Write definitions are also provided as a mapping.

            path=>vm_path         Map the given path for writes.

            Note that these is always an implicit write path,
            which is a temporary directory for the instance.

                /=>temp_dir
        """
        if vcpus is None:
            vcpus = 1
        if pack is None:
            pack = []
        if read is None:
            read = []

        args = ["novmm"]
        devices = []

        # Choose the latest kernel by default.
        if kernel is None:
            available_kernels = self._kernels.list()
            if len(available_kernels) > 0:
                kernel = available_kernels[0]

        # Is our kernel valid?
        if kernel not in self._kernels.list():
            raise Exception("Kernel not found!")

        # Always add control sockets.
        ctrl = control.Control(os.getpid(), bind=True)
        args.extend(["-controlfd=%d" % ctrl.fd()])

        # Always add basic devices.
        devices.append(basic.Bios())
        devices.append(basic.Acpi())

        # Add the kernel arguments.
        # (Including the packed modules).
        args.extend(["-vmlinux", self._kernels.file(kernel, "vmlinux")])
        args.extend(["-sysmap", self._kernels.file(kernel, "sysmap")])
        args.extend(["-initrd", self._kernels.file(kernel, "initrd")])
        args.extend(["-setup", self._kernels.file(kernel, "setup")])
        release = open(self._kernels.file(kernel, "release")).read().strip()

        # Use uart devices?
        if com1:
            devices.append(serial.Com1())
        if com2:
            devices.append(serial.Com2())

        # ALways enable an RTC.
        devices.append(clock.Rtc())

        # Use a PCI bus?
        if not(nopci):
            devices.append(pci.PciBus())
            devices.append(pci.PciHostBridge())

        # Always enable the console.
        # The noguest binary that executes inside
        # the guest will use this as an RPC mechanism.
        devices.append(serial.Console(index=0, pci=not(nopci)))

        # Build our NICs.
        devices.extend([
            net.Nic(index=1+index, pci=not(nopci),
                **dict([
                opt.split("=", 1)
                for opt in nic.split(",")
            ]))
            for (index, nic) in zip(range(len(nic)), nic)
        ])

        # Build our disks.
        devices.extend([
            block.Disk(index=1+len(nic)+index, pci=not(nopci),
                **dict([
                opt.split("=", 1)
                for opt in disk.split(",")
            ]))
            for (index, disk) in zip(range(len(disk)), disk)
        ])

        # Add modules.
        if os.path.exists(self._kernels.file(kernel, "modules")):
            read.append(
                "/lib/modules/%s=>%s" % (
                    release,
                    self._kernels.file(kernel, "modules")
                )
            )

        # Add our packs.
        # NOTE: All packs are given relative to root.
        for p in pack:
            read.append(self._packs.file(p))

        # The root filesystem.
        devices.append(fs.FS(
            pci=not(nopci),
            tag="root",
            read=read,
            write=write))

        # Our init.
        devices.append(fs.FS(
            pci=not(nopci),
            tag="init",
            read=["/init=>%s" % utils.libexec("noguest")]
        ))

        # Enable user-memory.
        devices.append(memory.UserMemory(
            size=1024*1024*(memsize or 1024)))

        # Save metadata.
        info = {
            "name": name,
            "vcpus": vcpus,
            "kernel": kernel,
            "devices": [
                (dev.__class__.__name__, dev.info())
                for dev in devices if dev
            ],
        }
        self._instances.add(str(os.getpid()), info)

        # Provide our CPU data.
        args.append("-vcpus=%s" %
            json.dumps([
                cpu.Cpu().arg() for _ in range(vcpus)
        ]))

        # Construct our cmdline.
        args.append("-cmdline=%s %s" % (" ".join([
            dev.cmdline()
            for dev in devices
            if dev.cmdline() is not None
        ]), cmdline))

        # Execute the instance.
        args.append("-devices=%s" % json.dumps([
            dev.arg() for dev in devices
        ]))
        args.extend(["-%s" % x for x in vmmopt])

        sys.stderr.write("exec: %s\n" % " ".join(args))
        os.execv(utils.libexec("novmm"), args)

    def execute(self,
            id=cli.StrOpt("The instance id."),
            name=cli.StrOpt("The instance name."),
            *command):

        """ Execute a command inside a novm. """

        if id is not None:
            if name is not None:
                raise Exception("Id must be specified alone.")
            obj_id = id
        else:
            obj_id = self._instances.find(name=name)
        if obj_id is None or not obj_id in self._instances.list():
            raise Exception("Instance not found.")

        ctrl = control.Control(obj_id, bind=False)
        ctrl.execute(command)

    def list(self,
            devices=cli.BoolOpt("Include device info?")):
        """ List running instances. """
        legit_instances = []
        for instance in self._instances.list():
            try:
                os.kill(int(instance), 0)
            except OSError:
                self._instances.remove(instance)
                continue
            legit_instances.append(instance)
        rval = self._instances.show()
        if not devices:
            for value in rval.values():
                if "devices" in value:
                    del value["devices"]
        return rval

    def packs(self):
        """ List available packs. """
        return self._packs.show()

    def getpack(self,
            url=cli.StrOpt("The pack URL (e.g. file: or http:)."),
            nocache=cli.BoolOpt("Don't use a cached version."),
            name=cli.StrOpt("A user-provided name.")):
        """ Fetch a new pack. """
        if not nocache:
            obj_id = self._packs.find(url=url, name=name)
            if obj_id is not None:
                return obj_id
        if url is None:
            raise Exception("Need URL.")
        return self._packs.fetch(url, name=name)

    def mkpack(self,
            output=cli.StrOpt("The output file."),
            path=cli.StrOpt("The input path."),
            exclude=cli.ListOpt("Subpaths to exclude."),
            include=cli.ListOpt("Subpaths to include.")):
        """ Create a pack from a tree. """
        if output is None:
            output = tempfile.mktemp()
        if path is None:
            path = os.getcwd()
        utils.packdir(path, output, exclude=exclude, include=include)
        return "file://%s" % os.path.abspath(output)

    def rmpack(self,
            id=cli.StrOpt("The pack id."),
            name=cli.StrOpt("The pack name."),
            url=cli.StrOpt("The pack URL")):

        """ Remove an existing pack. """

        if id is not None:
            if name is not None or url is not None:
                raise Exception("Id must be specified alone.")
            obj_id = id
        else:
            obj_id = self._packs.find(name=name, url=url)
        if obj_id is None or not obj_id in self._packs.list():
            raise Exception("Pack not found.")
        self._packs.remove(obj_id)

    def kernels(self):
        """ List available kernels. """
        return self._kernels.show()

    def getkernel(self,
            url=cli.StrOpt("The kernel URL (e.g. file: or http:)."),
            nocache=cli.BoolOpt("Don't use a cached version."),
            name=cli.StrOpt("A user-provided name.")):
        """ Fetch a new kernel. """
        if not nocache:
            obj_id = self._kernels.find(url=url, name=name)
            if obj_id is not None:
                return obj_id
        if url is None:
            raise Exception("Need URL.")
        return self._kernels.fetch(url, name=name)

    def mkkernel(self,
            output=cli.StrOpt("The output file."),
            release=cli.StrOpt("The kernel release (automated)."),
            modules=cli.StrOpt("Path to the kernel modules."),
            vmlinux=cli.StrOpt("Path to the vmlinux file."),
            bzimage=cli.StrOpt("Path to the compressed image."),
            setup=cli.StrOpt("Path to the setup header."),
            sysmap=cli.StrOpt("Path to the system map.")):

        """ Make a new kernel from an local kernel. """

        if output is None:
            output = tempfile.mktemp()

        # Find the files for this kernel.
        if release is None:
            release = platform.uname()[2]
        if modules is None:
            modules = "/lib/modules/%s" % release
        if bzimage is None:
            bzimage = "/boot/vmlinuz-%s" % release
        if vmlinux is None:
            vmlinux_file = tempfile.NamedTemporaryFile()
            subprocess.check_call(
                [utils.libexec("extract-vmlinux"), bzimage],
                stdout=vmlinux_file)
            vmlinux = vmlinux_file.name
        if setup is None:
            setup_file = tempfile.NamedTemporaryFile()
            setup_file.write(open(bzimage, 'r+b').read(4096))
            setup = setup_file.name
        if sysmap is None:
            sysmap = "/boot/System.map-%s" % release

        # Copy all the files into a single directory.
        temp_dir = tempfile.mkdtemp()
        try:
            # Make our initrd.
            subprocess.check_call(
                [utils.libexec("mkinitramfs"), release, modules],
                stdout=open(os.path.join(temp_dir, "initrd"), 'w'),
                close_fds=True)

            shutil.copy(vmlinux, os.path.join(temp_dir, "vmlinux"))
            shutil.copy(sysmap, os.path.join(temp_dir, "sysmap"))
            shutil.copy(setup, os.path.join(temp_dir, "setup"))
            shutil.copytree(modules, os.path.join(temp_dir, "modules"))
            os.makedirs(os.path.join(temp_dir, "modules"))
            open(os.path.join(temp_dir, "release"), "w").write(release)
            utils.packdir(temp_dir, output)
            return "file://%s" % os.path.abspath(output)
        finally:
            shutil.rmtree(temp_dir)

    def rmkernel(self,
            id=cli.StrOpt("The kernel id."),
            name=cli.StrOpt("The kernel name."),
            url=cli.StrOpt("The kernel URL")):

        """ Remove an existing kernel. """

        if id is not None:
            if name is not None or url is not None:
                raise Exception("Id must be specified alone.")
            obj_id = id
        else:
            obj_id = self._kernels.find(name=name, url=url)
        if obj_id is None or not obj_id in self._kernels.list():
            raise Exception("Kernel not found.")
        self._kernels.remove(obj_id)
