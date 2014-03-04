"""
Basic devices.
"""
from . import device

class Bios(device.Device):

    driver = "bios"

    def cmdline(self):
        return "intel_pstate=disable"

class Acpi(device.Device):

    driver = "acpi"
