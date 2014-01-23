"""
Filesystem device functions.
"""
import uuid
import tempfile
import shutil

from . import utils
from . import virtio

class FS(virtio.Device):

    virtio_driver = "fs"

    def __init__(
            self,
            tag=None,
            read=None,
            write=None,
            **kwargs):

        super(FS, self).__init__(**kwargs)

        if tag is None:
            tag = str(uuid.uuid4())
        if read is None:
            read = []
        if write is None:
            write = []

        # Save our tag.
        self._tag = tag

        # Append our read mapping.
        self._read = {'/': []}
        for path in read:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                self._read['/'].append(path)
            else:
                if not spec[0] in self._read:
                    self._read[spec[0]] = []
                self._read[spec[0]].append(spec[1])

        # Append our write mapping.
        tempdir = tempfile.mkdtemp()
        utils.cleanup(shutil.rmtree, tempdir)
        self._write = {'/': tempdir}

        for path in write:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                self._write['/'] = path
            else:
                self._write[spec[1]] = spec[0]

    def data(self):
        return {
            "read": self._read,
            "write": self._write,
            "tag": self._tag,
        }
