"""
PCI functionality.
"""
from . import device

class PciBus(device.Device):

    driver = "pci-bus"

    def cmdline(self):
        return "pci=conf1"

class PciHostBridge(device.Device):

    # NOTE: For now, PCI support is pretty sketchy.
    # Generally, we'll need to have a hostbridge appear
    # in the list of devices. This is all voodoo to me,
    # so for now the problem is solved quite simply by
    # passing a command line parameter above that forces
    # appropriate PCI detection.

    driver = "pci-hostbridge"
