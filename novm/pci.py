"""
PCI functionality.
"""
from . import device

class PciBus(device.Device):

    def device(self):
        return super(PciBus, self)._device(driver="pci-bus")

    def info(self):
        return None
