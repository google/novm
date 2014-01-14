"""
Basic devices.
"""
from . import device

class Bios(device.Device):

    driver = "bios"

class Acpi(device.Device):

    driver = "acpi"
