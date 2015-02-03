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
import json
import tempfile

from . import cli
from . import manager
from . import exceptions

class NovmShell(object):

    def __init__(self, *args, **kwargs):
        self._manager = manager.NovmManager(*args, **kwargs)

    def create(self,
            name=cli.StrOpt("The instance name."),
            cpus=cli.IntOpt("The number of vcpus."),
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
            env=cli.ListOpt("Specify an environment variable."),
            cwd=cli.StrOpt("The process working directory."),
            terminal=cli.BoolOpt("Change the terminal mode."),
            *command):

        """
        Run a new instance.

        Network definitions are provided as --nic [opt=val],...

            Available options are:

            mac=00:11:22:33:44:55 Set the MAC address.
            tapname=tap1          Set the tap name.
            bridge=br0            Enslave to a bridge.
            ip=192.168.1.2/24     Set the IP address.
            gateway=192.168.1.1   Set the gateway IP.
            debug=true            Enable debugging.

        Disk definitions are provided as --disk [opt=val],...

            Available options are:

            filename=disk         Set the backing file (raw).
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
        if cpus is None or isinstance(cpus, cli.IntOpt):
            cpus = 1
        if memsize is None or isinstance(memsize, cli.IntOpt):
            memsize = 1024
        if pack is None or isinstance(pack, cli.ListOpt):
            pack = []
        if not read or isinstance(read, cli.ListOpt):
            read = ["/"]

        return self._manager.create(
            name=name,
            cpus=cpus,
            memsize=memsize,
            kernel=kernel,
            init=init,
            nics=nic,
            disks=disk,
            repos=repo,
            read=read,
            write=write,
            nopci=nopci,
            com1=com1,
            com2=com2,
            cmdline=cmdline,
            vmmopt=vmmopt,
            nofork=nofork,
            env=env,
            cwd=cwd,
            terminal=terminal,
            command=command)

    @cli.alwaysjson
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
            raise exceptions.CommandInvalid()

        split_args = [arg.split("=", 1) for arg in command[1:]]
        if [arg for arg in split_args if len(arg) != 2]:
            raise Exception("Arguments should be key=value.")
        kwargs = dict(split_args)

        norm_kwargs = dict()
        for (k, v) in list(kwargs.items()):
            # Deserialize as JSON object.
            # This gives us strings, integers,
            # dictionaries, lists, etc. for free.
            norm_kwargs[k] = json.loads(v)

        return self._manager.rpc(
            id=id,
            name=name,
            command=command[0],
            args=norm_kwargs)

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

        return self._manager.run_noguest(
            id=id,
            name=name,
            env=env,
            cwd=cwd,
            terminal=terminal,
            command=command)

    def clean(self,
            id=cli.StrOpt("The instance id."),
            name=cli.StrOpt("The instance name.")):

        """ Remove stale instance information. """
        return self._manager.clean(id=id, name=name)

    def cleanall(self):

        """ Remove everything not alive. """
        return self._manager.cleanall()

    def list(self,
            full=cli.BoolOpt("Include device info?"),
            alive=cli.BoolOpt("Include only alive instances?")):

        """ List running instances. """
        rval = self._manager.list(alive=alive)

        if not full:
            # Prune devices, etc. unless requested.
            for value in list(rval.values()):
                for k in ("kernel", "devices", "vcpus"):
                    if k in value:
                        del value[k]

        return rval

    def packs(self):

        """ List available packs. """
        return self._manager._packs.show()

    def getpack(self,
            url=cli.StrOpt("The pack URL (e.g. file: or http:)."),
            nocache=cli.BoolOpt("Don't use a cached version."),
            name=cli.StrOpt("A user-provided name.")):

        """ Fetch a new pack. """
        return self._manager.getpack(url=url, nocache=nocache, name=name)

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

        return self._manager.mkpack(
            output=output,
            id=id,
            name=name,
            path=path,
            exclude=exclude,
            include=include)

    def rmpack(self,
            id=cli.StrOpt("The pack id."),
            name=cli.StrOpt("The pack name."),
            url=cli.StrOpt("The pack URL")):

        """ Remove an existing pack. """
        return self._manager.rmpack(id=id, name=name, url=url)

    def kernels(self):

        """ List available kernels. """
        return self._manager.kernels()

    def getkernel(self,
            url=cli.StrOpt("The kernel URL (e.g. file: or http:)."),
            nocache=cli.BoolOpt("Don't use a cached version."),
            name=cli.StrOpt("A user-provided name.")):

        """ Fetch a new kernel. """
        return self._manager.getkernel(url=url, nocache=nocache, name=name)

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
        if output is None:
            output = tempfile.mktemp()

        return self._manager.mkkernel(
            output=output,
            release=release,
            modules=modules,
            vmlinux=vmlinux,
            bzimage=bzimage,
            setup=setup,
            sysmap=sysmap,
            nomodules=nomodules)

    def rmkernel(self,
            id=cli.StrOpt("The kernel id."),
            name=cli.StrOpt("The kernel name."),
            url=cli.StrOpt("The kernel URL")):

        """ Remove an existing kernel. """
        self._manager.rmkernel(id=id, name=name, url=url)
