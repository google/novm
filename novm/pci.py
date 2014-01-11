"""
PCI functionality.
"""
from . import device

class PciBus(device.Device):

    driver = "pci-bus"
