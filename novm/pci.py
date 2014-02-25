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
    # in the list of devices.
    # For information on the bridge that might normally
    # appear, see src/novmm/machine/pcihost.go.

    driver = "pci-hostbridge"
