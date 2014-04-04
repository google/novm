"""
Block device functions.
"""
from . import virtio

class Disk(virtio.Driver):

    """ A Virtio block device. """

    virtio_driver = "block"

    def create(self,
            index=0,
            filename=None,
            dev=None,
            **kwargs):

        if filename is None:
            filename = "/dev/null"
        if dev is None:
            dev = "vd" + chr(ord("a") + index)

        # Open the device.
        f = open(filename, 'r+b')

        return super(Disk, self).create(data={
                "dev": dev,
                "fd": os.dup(f.fileno()),
            }, **kwargs)

virtio.Driver.register(Disk)
