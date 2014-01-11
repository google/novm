"""
Basic devices.
"""
from . import device

class Bios(device.Device):

    driver = "bios"

class Tss(device.Device):

    driver = "tss"

class Acpi(device.Device):

    driver = "acpi"
