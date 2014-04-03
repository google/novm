"""
Timers, etc.
"""
from . import device

class Rtc(device.Driver):

    driver = "rtc"

device.Driver.register(Rtc)
