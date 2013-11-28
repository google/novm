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

    def info(self):
        raise NotImplementedError()

    def _device(self, driver, data=None):
        return super(Device, self)._device(
            driver="virtio-%s-%s" % (
                self._pci and "pci" or "mmio", driver),
            data=data
        )
