"""
Basic devices.
"""
from . import device

class Bios(device.Device):

    driver = "bios"

class Acpi(device.Device):

    driver = "acpi"

    def cmdline(self):
        # Our ACPI implementation is currently
        # quite broken (we don't even have a DSDT).
        # Therefore, to stop Linux from complaining,
        # we intentionally disable power-states.
        return "intel_pstate=disable"

class Apic(device.Device):

    driver = "apic"

class Pit(device.Device):

    driver = "pit"
