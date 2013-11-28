"""
Device specification.
"""

from . import utils

class Device(object):

    def __init__(
            self,
            name=None,
            debug=False):

        super(Device, self).__init__()

        # Do we have a name?
        self._name = name

        # Debugging?
        self._debug = utils.asbool(debug)

    def info(self):
        raise NotImplementedError()

    def cmdline(self):
        return None

    def _device(self, driver, data=None):
        if data is None:
            data = {}
        return {
            "driver": driver,
            "name": self._name or driver,
            "debug": self._debug,
            "data": data,
        }

    def device(self):
        raise NotImplementedError()
