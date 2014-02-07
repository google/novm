"""
Filesystem device functions.
"""
import os
import uuid
import tempfile
import shutil

from . import utils
from . import virtio
from . import docker

class FS(virtio.Device):

    """ Virtio Filesystem (plan9) """

    virtio_driver = "fs"

    def __init__(
            self,
            tag=None,
            tempdir=None,
            read=None,
            write=None,
            dockerdb=None,
            **kwargs):

        super(FS, self).__init__(**kwargs)

        if tag is None:
            tag = str(uuid.uuid4())
        if read is None:
            read = []
        if write is None:
            write = []
        if tempdir is None:
            tempdir = tempfile.mkdtemp()
            utils.cleanup(shutil.rmtree, tempdir)
        if not os.path.exists(tempdir):
            os.makedirs(tempdir)

        # Save our tag.
        self._tag = tag

        # Append our read mapping.
        self._read = {'/': []}
        for path in read:
            # Do we support docker containers?
            # We accept arguments in the form:
            #  docker:<repository[:tag]>[,key=value]
            if dockerdb is not None and path.startswith("docker:"):
                args = path[7:].split(",")
                repository = args[0]
                clientargs = dict([arg.split("=", 1) for arg in args[1:]])
                client = docker.RegistryClient(dockerdb, **clientargs)
                self._read['/'].extend(client.pull_repository(repository))

            else:
                spec = path.split("=>", 1)
                if len(spec) == 1:
                    self._read['/'].append(path)
                else:
                    if not spec[0] in self._read:
                        self._read[spec[0]] = []
                    self._read[spec[0]].append(spec[1])

        # Append our write mapping.
        self._write = {'/': tempdir}

        for path in write:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                self._write['/'] = path
            else:
                self._write[spec[0]] = spec[1]

    def data(self):
        return {
            "read": self._read,
            "write": self._write,
            "tag": self._tag,
        }
