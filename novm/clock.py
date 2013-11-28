"""
Console functions.
"""
from . import device

class Rtc(device.Device):

    def device(self):
        return super(Rtc, self)._device(driver="rtc")

    def info(self):
        return None
