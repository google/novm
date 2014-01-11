"""
Device specification.
"""

from . import utils

class Device(object):

    debug = False

    def __init__(
            self,
            name=None,
            debug=False):

        super(Device, self).__init__()

        # Do we have a name?
        self._name = name

        # Debugging?
        self._debug = self.debug or utils.asbool(debug)

    def info(self):
        """ User-displayed device info. """
        return None

    def cmdline(self):
        """ Return a Linux cmdline paramter. """
        return None

    @property
    def driver(self):
        """ Return the driver for novmm. """
        raise NotImplementedError()

    def data(self):
        """ Device data (encoded on startup). """
        return None

    def arg(self):
        """ Returns the full device specification. """
        return {
            "driver": self.driver,
            "name": self._name or self.driver,
            "debug": self._debug,
            "data": self.data(),
        }
