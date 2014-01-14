"""
Memory devices.
"""
import os
import tempfile

from . import device

class UserMemory(device.Device):

    driver = "user-memory"

    def __init__(
            self,
            size=None,
            fd=None,
            **kwargs): 

        super(UserMemory, self).__init__(**kwargs)

        # No file given?
        if fd is None:
            with tempfile.NamedTemporaryFile() as tf:
                fd = os.dup(tf.fileno())

        # No size given? Default to file size.
        if size is None:
            fd_stat = os.fstat(fd)
            size = fd_stat.st_size

        # Truncate the file.
        os.ftruncate(fd, size)

        # Save our data.
        self._fd = fd
        self._size = size

    def data(self):
        return {
            "fd": self._fd,
        }

    def info(self):
        return self._size

class Com1(device.Device):

    driver = "uart"

    def data(self):
        return {
            "address": 0x3f8,
            "interrupt": 4,
        }

    def cmdline(self):
        return "console=uart,io,0x3f8"

    def info(self):
        return "com1"

class Com2(device.Device):

    driver = "uart"

    def data(self):
        return {
            "address": 0x2f8,
            "interrupt": 3,
        }

    def cmdline(self):
        return "console=uart,io,0x2f8"

    def info(self):
        return "com2"
