"""
Virtio device specification.
"""
from . import device

class Device(device.Device):

    def __init__(
            self,
            pci=False,
            **kwargs):

        super(Device, self).__init__(**kwargs)

        # PCI?
        self._pci = pci

    @property
    def driver(self):
        if self._pci:
            return "virtio-pci-%s" % self.virtio_driver
        else:
            return "virtio-mmio-%s" % self.virtio_driver
