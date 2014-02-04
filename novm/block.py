"""
Block device functions.
"""

from . import virtio

class Disk(virtio.Device):

    """ A Virtio block device. """

    virtio_driver = "block"

    def __init__(
            self,
            index=0,
            filename=None,
            dev=None,
            **kwargs):

        super(Disk, self).__init__(index=index, **kwargs)

        if filename is None:
            filename = "/dev/null"
        if dev is None:
            dev = "vd" + chr(ord("a") + index)

        # Save our arguments.
        self._info = {
            "dev": dev,
            "filename": filename,
        }

        # Open the device.
        self._file = open(filename, 'r+b')

    def data(self):
        return {
            "dev": self._info["dev"],
            "fd": self._file.fileno(),
        }

    def info(self):
        return self._info
