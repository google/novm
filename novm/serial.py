"""
Console functions.
"""
import os
import socket
import shutil

from . import utils
from . import device
from . import virtio

class Console(virtio.Device):

    def __init__(
            self,
            path=None,
            **kwargs): 

        super(Console, self).__init__(**kwargs)

        if path is None:
            path = "/var/run/%s.sock" % os.getpid()

        # Save our arguments.
        self._info = {
            "path": path,
        }

        # Bind the socket.
        self._sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        if os.path.exists(path):
            os.remove(path)
        utils.cleanup(shutil.rmtree, path)
        self._sock.bind(path)
        self._sock.listen(1)

    def device(self):
        return super(Console, self)._device(
            driver="console",
            data={
                "fd": self._sock.fileno(),
            },
        )

    def info(self):
        return self._info

class Com1(device.Device):

    def device(self):
        return super(Com1, self)._device(
            driver="uart",
            data={
                "address": 0x3f8,
                "interrupt": 4,
            },
        )

    def cmdline(self):
        return "console=uart,io,0x3f8"

    def info(self):
        return "com1"

class Com2(device.Device):

    def device(self):
        return super(Com2, self)._device(
            driver="uart",
            data={
                "address": 0x2f8,
                "interrupt": 3,
            },
        )

    def cmdline(self):
        return "console=uart,io,0x2f8"

    def info(self):
        return "com2"
