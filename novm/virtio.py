"""
Virtio device specification.
"""
from . import device

class Device(device.Device):

    def __init__(
            self,
            index=0,
            pci=False,
            **kwargs):

        super(Device, self).__init__(**kwargs)

        # PCI?
        self._pci = pci

        # Are we an MMIO device?
        # NOTE: We arbitrarily pick 0xeXXXXXXX as the
        # start for all of our virtio devices. If we 
        # have to do anymore reservation for I/O devices,
        # we might want to consider implemented something
        # a bit more thorough here.
        if not pci:
            self._index = index
            self._address = (0xe0000000 + index*4096)
            self._interrupt = 32 + index

    @property
    def driver(self):
        if self._pci:
            return "virtio-pci-%s" % self.virtio_driver
        else:
            return "virtio-mmio-%s" % self.virtio_driver

    def cmdline(self):
        if self._pci:
            return None
        else:
            return "virtio-mmio.%d@0x%x:%d:%d" % (
                self._index,
                self._address,
                self._interrupt,
                self._index)

    def data(self):
        if self._pci:
            return None
        else:
            return {
                "address": self._address,
                "interrupt": self._interrupt,
            }
