"""
Filesystem device functions.
"""
import os
import uuid
import tempfile
import shutil

from . import utils
from . import virtio

class FS(virtio.Driver):

    """ Virtio Filesystem (plan9) """

    virtio_driver = "fs"

    def create(self,
            tag=None,
            tempdir=None,
            read=None,
            write=None,
            fdlimit=None,
            **kwargs):

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

        # Append our read mapping.
        read_map = {'/': []}
        for path in read:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                read_map['/'].append(path)
            else:
                if not spec[0] in read_map:
                    read_map[spec[0]] = []
                read_map[spec[0]].append(spec[1])

        # Append our write mapping.
        write_map = {'/': tempdir}

        for path in write:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                write_map['/'] = path
            else:
                write_map[spec[0]] = spec[1]

        # Create our device.
        return super(FS, self).create(data={
            "read": read_map,
            "write": write_map,
            "tag": tag,
            "fdlimit": fdlimit or 0,
        }, **kwargs)

virtio.Driver.register(FS)
