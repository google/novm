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
import fcntl
import pickle
import tempfile

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
from . import docker
from . import exceptions

class NovmManager(object):

    def __init__(self, root=None):
        if root is None:
            self._root = os.getenv(
                "NOVM_ROOT",
                os.path.join(
                    os.getenv("HOME", tempfile.gettempdir()),
                    ".novm"))

        self._instances = db.Nodb(
            os.getenv(
                "NOVM_INSTANCES",
                os.path.join(self._root, "instances")))

        self._packs = db.Nodb(
            os.getenv(
                "NOVM_PACKS",
                os.path.join(self._root, "packs")))

        self._kernels = db.Nodb(
            os.getenv(
                "NOVM_KERNELS",
                os.path.join(self._root, "kernels")))

        # Our docker images.
        # This is cached because docker images
        # will form a tree. We essentially build
        # a list of paths for every docker repo
        # that the user specifies.
        self._docker = db.Nodb(
            os.getenv(
                "NOVM_DOCKER",
                os.path.join(self._root, "docker")))

        self._controls = os.path.join(self._root, "control")

    def create(self,
            name=cli.StrOpt("The instance name."),
            vcpus=cli.IntOpt("The number of vcpus."),
            memsize=cli.IntOpt("The member size (in mb)."),
            kernel=cli.StrOpt("The kernel to use."),
            init=cli.BoolOpt("Use a real init?"),
            nic=cli.ListOpt("Define a network device."),
            disk=cli.ListOpt("Define a block device."),
            pack=cli.ListOpt("Use a given read pack."),
            repo=cli.ListOpt("Use a docker repository."),
            read=cli.ListOpt("Define a backing filesystem read tree."),
            write=cli.ListOpt("Define a backing filesystem write tree."),
            nopci=cli.BoolOpt("Disable PCI devices?"),
            com1=cli.BoolOpt("Enable COM1 UART?"),
            com2=cli.BoolOpt("Enable COM2 UART?"),
            cmdline=cli.StrOpt("Extra command line options?"),
            vmmopt=cli.ListOpt("Options to pass to novmm."),
            nofork=cli.BoolOpt("Don't fork into the background."),
            terminal=cli.BoolOpt("Change the terminal mode."),
            *command):

        """ 
        Run a new instance.

        Network definitions are provided as --nic [opt=val],...

            Available options are:

            name=name             Set the device name.
            mac=00:11:22:33:44:55 Set the MAC address.
            tap=tap1              Set the tap name.
            bridge=br0            Enslave to a bridge.
            ip=192.168.1.2/24     Set the IP address.
            gateway=192.168.1.1   Set the gateway IP.
            debug=true            Enable debugging.

        Disk definitions are provided as --disk [opt=val],...

            Available options are:

            name=name             Set the device name.
            file=filename         Set the backing file (raw).
            dev=vda               Set the device name.
            debug=true            Enable debugging.

        Read definitions are provided as a mapping.

            vm_path=>path         Map the given path for reads.

        Write definitions are also provided as a mapping.

            vm_path=>path         Map the given path for writes.

            Note that these is always an implicit write path,
            which is a temporary directory for the instance.

                /=>temp_dir

        Docker repositories are provided as follows.

            <repository[:tag]>[,key=value]

            You should specify at least username, password.
        """
        if vcpus is None or isinstance(vcpus, cli.IntOpt):
            vcpus = 1
        if pack is None or isinstance(pack, cli.ListOpt):
            pack = []
        if not read or isinstance(read, cli.ListOpt):
            read = ["/"]

        args = ["novmm"]
        devices = []

        if not nofork:
            r_pipe, w_pipe = os.pipe()
            child = os.fork()
            if child != 0:
                # Wait for the result,
                # Return the new instance_id.
                os.close(w_pipe)
                r = os.fdopen(r_pipe, 'r')
                data = r.read()
                if not data:
                    # Closed by exec().
                    # At this point we proceed, either to
                    # run a command or to give back the pid.
                    if command:
                        run_cmd = [child, None, None, None, terminal]
                        run_cmd.extend(command)
                        return self.run(*run_cmd)
                    else:
                        return child
                else:
                    # This is a pickle'd exception.
                    (exc_type, exc_value) = pickle.loads(data)
                    while True:
                        (pid, status) = os.waitpid(child, 0)
                        if pid == child and os.WIFEXITED(status):
                            raise exc_value
            else:
                # Are we running a command?
                # Make sure this exits when we're done.
                if command:
                    utils.cleanup()

                # Continue to create the VM.
                # The read pipe will be closed automatically
                # only when the new VM is actually running.
                os.close(r_pipe)
                fcntl.fcntl(w_pipe, fcntl.F_SETFD, fcntl.FD_CLOEXEC)

        try:
            # Choose the latest kernel by default.
            if kernel is None or isinstance(kernel, StrOpt):
                available_kernels = self._kernels.list()
                if len(available_kernels) > 0:
                    kernel = available_kernels[0]

            # Is our kernel valid?
            if kernel not in self._kernels.list():
                raise Exception("Kernel not found!")

            # Are we using a real init?
            if init:
                args.extend(["-init"])

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

            # Always enable an RTC.
            devices.append(clock.Rtc())

            # Use a PCI bus?
            if not(nopci):
                devices.append(pci.PciBus())

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

            # Add our docker repositories.
            # NOTE: These are also all relative to root.
            for d in repo:
                opts = d.split(",")
                repository = opts[0]
                kwargs = dict([arg.split("=", 1) for arg in opts[1:]])
                client = docker.RegistryClient(self._docker, **kwargs)
                read.extend(client.pull_repository(repository))

            # The root filesystem.
            devices.append(fs.FS(
                index=1+len(nic)+len(disk),
                pci=not(nopci),
                tag="root",
                tempdir=self._instances.file(str(os.getpid())),
                read=read,
                write=write))

            # Add noguest as our init.
            devices.append(fs.FS(
                index=1+len(nic)+len(disk)+1,
                pci=not(nopci),
                tag="init",
                read=["/init=>%s" % utils.libexec("noguest")]
            ))

            # Enable user-memory.
            devices.append(memory.UserMemory(
                size=1024*1024*(memsize or 1024)))

            # Provide our CPU data.
            args.append("-vcpus=%s" %
                json.dumps([
                    cpu.Cpu().arg() for _ in range(vcpus)
            ]))

            # Add our control socket.
            ctrl_path = os.path.join(self._controls, "%s.ctrl" % str(os.getpid()))
            ctrl = control.Control(ctrl_path, bind=True)
            args.extend(["-controlfd=%d" % ctrl.fd()])

            # Construct our cmdline.
            args.append("-cmdline=%s %s" % (" ".join([
                dev.cmdline()
                for dev in devices
                if dev.cmdline() is not None
            ]), cmdline))
            args.append("-devices=%s" % json.dumps([
                dev.arg() for dev in devices
            ]))
            args.extend(["-%s" % x for x in vmmopt])

            # Save metadata.
            info = {
                "name": name,
                "vcpus": vcpus,
                "kernel": kernel,
                "devices": [
                    (dev.__class__.__name__, dev.info())
                    for dev in devices
                    if dev.info()
                ],
                "ips": [
                    dev.ip()
                    for dev in devices
                    if isinstance(dev, net.Nic) and dev.ip()
                ]
            }
            self._instances.add(str(os.getpid()), info)
            utils.cleanup(self._instances.remove, str(os.getpid()))

            # Close off final descriptors.
            if not nofork:
                null_w = open("/dev/null", "w")
                null_r = open("/dev/null", "r")
                os.dup2(null_r.fileno(), 0)
                os.dup2(null_w.fileno(), 1)
                os.dup2(null_w.fileno(), 2)

            # Execute our VMM.
            os.execv(utils.libexec("novmm"), args)

        except (SystemExit, KeyboardInterrupt):
            raise
        except:
            if not nofork:
                # Write our exception.
                w = os.fdopen(w_pipe, 'w')
                pickle.dump(sys.exc_info()[:2], w)
                w.close()
            # Raise in the main thread.
            exc_info = sys.exc_info()
            raise exc_info[0], exc_info[1], exc_info[2]

    def control(self,
            id=cli.StrOpt("The instance id."),
            name=cli.StrOpt("The instance name."),
            *command):

        """
        Execute a control command.

        Available commands depend on the VMM.

        For example:

            To pause the first VCPU:

                vcpu id=0 paused=true

            To enable tracing:

                trace enable=true
        """
        if len(command) == 0:
            raise Exception("Need to provide a command.")

        split_args = [arg.split("=", 1) for arg in command[1:]]
        if [arg for arg in split_args if len(arg) != 2]:
            raise Exception("Arguments should be key=value.")
        kwargs = dict(split_args)

        norm_kwargs = dict()
        for (k, v) in kwargs.items():
            # Deserialize as JSON object.
            # This gives us strings, integers,
            # dictionaries, lists, etc. for free.
            norm_kwargs[k] = json.loads(v)

        obj_id = self._instances.find(obj_id=id, name=name)

        ctrl_path = os.path.join(self._controls, "%s.ctrl" % obj_id)
        ctrl = control.Control(ctrl_path, bind=False)

        return ctrl.rpc(command[0], **norm_kwargs)

    def run(self,
            id=cli.StrOpt("The instance id."),
            name=cli.StrOpt("The instance name."),
            env=cli.ListOpt("Specify an environment variable."),
            cwd=cli.StrOpt("The process working directory."),
            terminal=cli.BoolOpt("Change the terminal mode."),
            *command):

        """ Execute a command inside a novm. """
        if env is not None and len(env) == 0:
            env = None
        if len(command) == 0:
            raise exceptions.CommandInvalid()

        obj_id = self._instances.find(obj_id=id, name=name)

        ctrl_path = os.path.join(self._controls, "%s.ctrl" % obj_id)
        ctrl = control.Control(ctrl_path, bind=False)

        return ctrl.run(command, env=env, cwd=cwd, terminal=terminal)

    def _is_alive(self, pid):
        """ Is this process still around? """
        return os.path.exists("/proc/%s" % str(pid))

    def clean(self,
            id=cli.StrOpt("The instance id."),
            name=cli.StrOpt("The instance name.")):

        """ Remove stale instance information. """
        self._instances.remove(obj_id=id, name=name)

    def cleanall(self):

        """ Remove everything not alive. """
        for pid in self._instances.list():
            # Is this process still around?
            if not self._is_alive(pid):
                self._instances.remove(obj_id=pid)

    def list(self,
            full=cli.BoolOpt("Include device info?"),
            alive=cli.BoolOpt("Include only alive instances?")):

        """ List running instances. """
        rval = self._instances.show()
        if not full:
            # Prune devices, etc. unless requested.
            for value in rval.values():
                for k in ("kernel", "devices", "vcpus"):
                    if k in value:
                        del value[k]
        for (pid, value) in rval.items():
            # Add information about liveliness.
            if self._is_alive(pid):
                value["alive"] = True
            else:
                value["alive"] = False
        if alive:
            # Filter alive instances.
            rval = dict([
                (k, v)
                for (k, v) in rval.items()
                if v.get("alive")
            ])
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
            try:
                # Try to find an existing pack.
                return self._packs.find(url=url, name=name)
            except KeyError:
                pass
        return self._packs.fetch(url, name=name)

    def mkpack(self,
            id=cli.StrOpt("The instance id."),
            name=cli.StrOpt("The instance name."),
            output=cli.StrOpt("The output file."),
            path=cli.StrOpt("The input path."),
            exclude=cli.ListOpt("Subpaths to exclude."),
            include=cli.ListOpt("Subpaths to include.")):
        """ Create a pack from a tree. """
        if output is None or isinstance(output, cli.StrOpt):
            output = tempfile.mktemp()
        if path is None or isinstance(path, cli.StrOpt):
            if id is not None or name is not None:
                # Is it an instance they want?
                obj_id = self._instances.find(obj_id=id, name=name)
                path = self._instances.file(obj_id)
            else:
                # Otherwise, use the current dir.
                path = os.getcwd()

        utils.packdir(path, output, exclude=exclude, include=include)
        return "file://%s" % os.path.abspath(output)

    def rmpack(self,
            id=cli.StrOpt("The pack id."),
            name=cli.StrOpt("The pack name."),
            url=cli.StrOpt("The pack URL")):

        """ Remove an existing pack. """
        self._packs.remove(obj_id=id, name=name, url=url)

    def kernels(self):
        """ List available kernels. """
        kernels = self._kernels.show()
        for (obj_id, data) in kernels.items():
            try:
                # Add the release.
                release = open(self._kernels.file(obj_id, "release")).read().strip()
                data["release"] = release
            except IOError:
                continue
        return kernels

    def getkernel(self,
            url=cli.StrOpt("The kernel URL (e.g. file: or http:)."),
            nocache=cli.BoolOpt("Don't use a cached version."),
            name=cli.StrOpt("A user-provided name.")):
        """ Fetch a new kernel. """
        if not nocache:
            try:
                # Try to find an existing kernel.
                return self._kernels.find(url=url, name=name)
            except KeyError:
                pass
        return self._kernels.fetch(url, name=name)

    def mkkernel(self,
            output=cli.StrOpt("The output file."),
            release=cli.StrOpt("The kernel release (automated)."),
            modules=cli.StrOpt("Path to the kernel modules."),
            vmlinux=cli.StrOpt("Path to the vmlinux file."),
            bzimage=cli.StrOpt("Path to the compressed image."),
            setup=cli.StrOpt("Path to the setup header."),
            sysmap=cli.StrOpt("Path to the system map."),
            nomodules=cli.BoolOpt("Don't include modules.")):

        """ Make a new kernel from an local kernel. """
        if output is None or isinstance(output, cli.StrOpt):
            output = tempfile.mktemp()

        # Find the files for this kernel.
        if release is None or isinstance(release, cli.StrOpt):
            release = platform.uname()[2]
        if modules is None or isinstance(modules, cli.StrOpt):
            modules = "/lib/modules/%s" % release
        if bzimage is None or isinstance(bzimage, cli.StrOpt):
            bzimage = "/boot/vmlinuz-%s" % release
        if vmlinux is None or isinstance(vmlinux, cli.StrOpt):
            vmlinux_file = tempfile.NamedTemporaryFile()
            subprocess.check_call(
                [utils.libexec("extract-vmlinux"), bzimage],
                stdout=vmlinux_file)
            vmlinux = vmlinux_file.name
        if setup is None or isinstance(setup, cli.StrOpt):
            setup_file = tempfile.NamedTemporaryFile()
            setup_file.write(open(bzimage, 'rb').read(4096))
            setup = setup_file.name
        if sysmap is None or isinstance(sysmap, cli.StrOpt):
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

            if nomodules:
                os.makedirs(os.path.join(temp_dir, "modules"))
            else:
                shutil.copytree(
                    modules,
                    os.path.join(temp_dir, "modules"),
                    ignore=shutil.ignore_patterns("source", "build"))

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
        self._kernels.remove(obj_id=id, name=name, url=url)
