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

    virtio_driver = "console"

class Com1(device.Device):

    driver = "uart"

    def data(self):
        return {
            "base": 0x3f8,
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
            "base": 0x2f8,
            "interrupt": 3,
        }

    def cmdline(self):
        return "console=uart,io,0x2f8"

    def info(self):
        return "com2"
