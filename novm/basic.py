"""
Basic devices.
"""
from . import device

class Bios(device.Driver):

    driver = "bios"

class Acpi(device.Driver):

    driver = "acpi"

    def create(self, **kwargs):
        # Our ACPI implementation is currently
        # quite broken (we don't even have a DSDT).
        # Therefore, to stop Linux from complaining,
        # we intentionally disable power-states.
        return super(Acpi, self).create(
            cmdline="intel_pstate=disable",
            **kwargs)

class Apic(device.Driver):

    driver = "apic"

class Pit(device.Driver):

    driver = "pit"

device.Driver.register(Bios)
device.Driver.register(Acpi)
device.Driver.register(Apic)
device.Driver.register(Pit)
