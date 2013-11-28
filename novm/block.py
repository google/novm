"""
Block device functions.
"""

from . import virtio

class Disk(virtio.Device):

    def __init__(
            self,
            index=0,
            filename=None,
            device=None,
            **kwargs):

        super(Disk, self).__init__(**kwargs)

        if filename is None:
            filename = "/dev/null"
        if device is None:
            device = "vd" + chr(ord("a") + index)

        # Save our arguments.
        self._info = {
            "device": device,
            "filename": filename,
        }

        # Open the device.
        self._file = open(filename, 'w+b')

    def device(self):
        return super(Disk, self)._device(
            driver="block",
            data={
                "device": self._info["device"],
                "fd": self._file.fileno(),
            },
        )

    def info(self):
        return self._info
