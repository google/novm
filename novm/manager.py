# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
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
import six

from . import utils
from . import db
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
            kernel=None,
            name=None,
            cpus=1,
            memsize=1024,
            init=False,
            nics=None,
            disks=None,
            packs=None,
            repos=None,
            read=None,
            write=None,
            nopci=False,
            com1=False,
            com2=False,
            cmdline=None,
            vmmopt=None,
            nofork=False,
            env=None,
            cwd=None,
            terminal=False,
            command=None):

        if nics is None:
            nics = []
        if disks is None:
            disks = []
        if packs is None:
            packs = []
        if repos is None:
            repos = []
        if read is None:
            read = []
        if write is None:
            write = []
        if vmmopt is None:
            vmmopt = []
        if cmdline is None:
            cmdline = ""
        else:
            cmdline += " "

        # Choose the latest kernel by default.
        if kernel is None:
            available_kernels = self._kernels.list()
            if len(available_kernels) > 0:
                kernel = available_kernels[0]

        # Our extra arguments.
        args = []

        # Are we using a real init?
        if init:
            args.extend(["-init"])

        # Is our kernel valid?
        if kernel not in self._kernels.list():
            raise Exception("Kernel not found!")

        # Add the kernel arguments.
        # (Including the packed modules).
        args.extend(["-vmlinux", self._kernels.file(kernel, "vmlinux")])
        args.extend(["-sysmap", self._kernels.file(kernel, "sysmap")])
        args.extend(["-initrd", self._kernels.file(kernel, "initrd")])
        args.extend(["-setup", self._kernels.file(kernel, "setup")])
        release = open(self._kernels.file(kernel, "release")).read().strip()

        # Prepare our device callback.
        # (This is done as a callback since run() may
        # fork, etc. and we may need to do certain setup
        # routines within the novmm process proper.
        def state(output):
            devices = []

            # Always add basic devices.
            devices.append(basic.Bios().create())
            devices.append(basic.Acpi().create())
            devices.append(basic.Apic().create())
            devices.append(basic.Pit().create())

            # Use uart devices?
            if com1:
                devices.append(serial.Uart().com1())
            if com2:
                devices.append(serial.Uart().com2())

            # Always enable an RTC.
            devices.append(clock.Rtc().create())

            # Use a PCI bus?
            if not(nopci):
                devices.append(pci.PciBus().create())

            # Enable user-memory.
            devices.append(memory.UserMemory().create(
                size=1024*1024*memsize))

            # Always enable the console.
            # The noguest binary that executes inside
            # the guest will use this as an RPC mechanism.
            devices.append(serial.Console().create(index=0, pci=not(nopci)))

            # Build our NICs.
            devices.extend([
                net.Nic().create(
                    index=1+index,
                    pci=not(nopci),
                    **dict([
                    opt.split("=", 1)
                    for opt in nic.split(",")
                ]))
                for (index, nic) in zip(list(range(len(nics))), nics)
            ])

            # Build our disks.
            devices.extend([
                block.Disk().create(
                    index=1+len(nics)+index,
                    pci=not(nopci),
                    **dict([
                    opt.split("=", 1)
                    for opt in odisk.split(",")
                ]))
                for (index, odisk) in zip(list(range(len(disks))), disks)
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
            for p in packs:
                read.append(self._packs.file(p))

            # Add our docker repositories.
            # NOTE: These are also all relative to root.
            for d in repos:
                opts = d.split(",")
                repository = opts[0]
                kwargs = dict([arg.split("=", 1) for arg in opts[1:]])
                client = docker.RegistryClient(self._docker, **kwargs)
                read.extend(client.pull_repository(repository))

            # The root filesystem.
            devices.append(fs.FS().create(
                index=1+len(nics)+len(disks),
                pci=not(nopci),
                tag="root",
                tempdir=self._instances.file(str(os.getpid())),
                read=read,
                write=write))

            # Add noguest as our init.
            devices.append(fs.FS().create(
                index=1+len(nics)+len(disks)+1,
                pci=not(nopci),
                tag="init",
                read=["/init=>%s" % utils.libexec("noguest")]
            ))

            # Create our vcpus.
            vcpus = [cpu.Cpu() for _ in range(cpus)]

            # Dump our generated state.
            json.dump({
                "vcpus": [vcpu.state() for vcpu in vcpus],
                "devices": [dev.state() for dev in devices],
            }, output)

            # Generate appropriate metadata.
            # NOTE: This is just convenience for
            # listing running novms, it doesn't have
            # any impact on the actual behaviour.
            metadata = {
                "name": name,
                "cpus": cpus,
                "memory": memsize,
                "kernel": kernel,
                "ips": [
                    dev.get("ip")
                    for dev in devices
                    if isinstance(dev, net.Nic) and dev.get("ip")
                ]
            }

            # Return our cmdline.
            # This is done this way because we can't
            # necessarily assume that the cmdline is
            # available until the devices are built,
            # but we can't build the devices until
            # we're in the novmm process (i.e. forked).
            args.append("-cmdline=%s%s" % (" ".join([
                dev.cmdline()
                for dev in devices
                if dev.cmdline() is not None
            ]), cmdline))

            return args, metadata

        # Execute the novmm process.
        return self.run_novmm(
            state,
            command=command,
            nofork=nofork,
            vmmopt=vmmopt,
            terminal=terminal,
            env=env,
            cwd=cwd)

    def run_novmm(self,
            state,
            command=None,
            nofork=False,
            vmmopt=None,
            **kwargs):

        if vmmopt is None:
            vmmopt = []

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
                        return self.run_noguest(command, id=child, **kwargs)
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
            args = ["novmm"]

            # Provide our state by dumping to a temporary file.
            state_file = tempfile.NamedTemporaryFile(mode='w', delete=False)
            os.remove(state_file.name)
            state_args, metadata = state(state_file)
            state_file.seek(0, 0)

            # Keep the descriptor.
            statefd = os.dup(state_file.fileno())
            utils.clear_cloexec(statefd)
            state_file.close()

            # Add our state.
            args.append("-statefd=%d" % statefd)
            args.extend(state_args)

            # Add our control socket.
            ctrl_path = os.path.join(self._controls, "%s.ctrl" % str(os.getpid()))
            ctrl = control.Control(ctrl_path, bind=True)
            args.extend(["-controlfd=%d" % ctrl.fd()])

            # Add extra (debugging) arguments.
            args.extend(["-%s" % x for x in vmmopt])

            # Add our metadata to the registry.
            self._instances.add(str(os.getpid()), metadata)
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
                w = os.fdopen(w_pipe, 'wb')
                pickle.dump(sys.exc_info()[:2], w, 2)
                w.close()
            # Raise in the main thread.
            exc_info = sys.exc_info()
            six.reraise(exc_info[0], exc_info[1], exc_info[2])

    def rpc(self, command, args, id=None, name=None):
        """ Execute a single control RPC. """
        obj_id = self._instances.find(obj_id=id, name=name)
        ctrl_path = os.path.join(self._controls, "%s.ctrl" % obj_id)
        ctrl = control.Control(ctrl_path, bind=False)
        return ctrl.rpc(command, **args)

    def run_noguest(self, command, id=None, name=None, **kwargs):
        """ Run a command inside the given guest. """
        obj_id = self._instances.find(obj_id=id, name=name)
        ctrl_path = os.path.join(self._controls, "%s.ctrl" % obj_id)
        ctrl = control.Control(ctrl_path, bind=False)
        return ctrl.run(command, **kwargs)

    def _is_alive(self, pid):
        """ Is this process still around? """
        return os.path.exists("/proc/%s" % str(pid))

    def clean(self, id=None, name=None):
        self._instances.remove(obj_id=id, name=name)

    def cleanall(self):
        for pid in self._instances.list():
            # Is this process still around?
            if not self._is_alive(pid):
                self._instances.remove(obj_id=pid)

    def list(self, alive=False):
        rval = self._instances.show()

        for (pid, value) in list(rval.items()):
            # Add information about liveliness.
            if self._is_alive(pid):
                value["alive"] = True
            else:
                value["alive"] = False

        if alive:
            # Filter alive instances.
            rval = dict([
                (k, v)
                for (k, v) in list(rval.items())
                if v.get("alive")
            ])

        return rval

    def packs(self):
        return self._packs.show()

    def getpack(self, url, nocache=False, name=None):
        if not nocache:
            try:
                # Try to find an existing pack.
                return self._packs.find(url=url, name=name)
            except KeyError:
                pass
        return self._packs.fetch(url, name=name)

    def mkpack(self,
            output,
            id=None,
            name=None,
            path=None,
            exclude=None,
            include=None):

        if path is None:
            if id is not None or name is not None:
                # Is it an instance they want?
                obj_id = self._instances.find(obj_id=id, name=name)
                path = self._instances.file(obj_id)
            else:
                # Otherwise, use the current dir.
                path = os.getcwd()

        utils.packdir(path, output, exclude=exclude, include=include)
        return "file://%s" % os.path.abspath(output)

    def rmpack(self, id=None, name=None, url=None):
        self._packs.remove(obj_id=id, name=name, url=url)

    def kernels(self):
        kernels = self._kernels.show()
        for (obj_id, data) in list(kernels.items()):
            try:
                # Add the release.
                release = open(self._kernels.file(obj_id, "release")).read().strip()
                data["release"] = release
            except IOError:
                continue
        return kernels

    def getkernel(self, url, nocache=False, name=None):
        if not nocache:
            try:
                # Try to find an existing kernel.
                return self._kernels.find(url=url, name=name)
            except KeyError:
                pass
        return self._kernels.fetch(url, name=name)

    def mkkernel(self,
            output,
            release=None,
            modules=None,
            vmlinux=None,
            bzimage=None,
            setup=None,
            sysmap=None,
            nomodules=False):

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
            setup_file = tempfile.NamedTemporaryFile(mode='wb')
            setup_file.write(open(bzimage, 'rb').read(4096))
            setup_file.flush()
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

    def rmkernel(self, id=None, name=None, url=None):
        self._kernels.remove(obj_id=id, name=name, url=url)
