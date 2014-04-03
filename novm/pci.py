"""
PCI functionality.
"""
from . import device

class PciBus(device.Driver):

    driver = "pci-bus"

    def create(self, **kwargs):
        return super(PciBus, self).create(
            cmdline="pci=conf1",
            **kwargs)

class PciHostBridge(device.Driver):

    # NOTE: For now, PCI support is pretty sketchy.
    # Generally, we'll need to have a hostbridge appear
    # in the list of devices.
    # For information on the bridge that might normally
    # appear, see src/novmm/machine/pcihost.go.

    driver = "pci-hostbridge"

device.Driver.register(PciBus)
device.Driver.register(PciHostBridge)
