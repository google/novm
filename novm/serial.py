"""
Console functions.
"""
from . import device
from . import virtio

class Console(virtio.Driver):

    """ A Virtio serial/console device. """

    virtio_driver = "console"

class Uart(device.Driver):

    driver = "uart"

    def com1(self, **kwargs):
        return self.create(data={
                "base": 0x3f8,
                "interrupt": 4,
            },
            cmdline="console=uart,io,0x3f8",
            **kwargs)

    def com2(self, **kwargs):
        return self.create(data={
                "base": 0x2f8,
                "interrupt": 3,
            },
            cmdline="console=uart,io,0x2f8",
            **kwargs)

virtio.Driver.register(Console)
device.Driver.register(Uart)
