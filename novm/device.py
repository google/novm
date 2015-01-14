# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""
Device specification.
"""
import uuid

from . import state
from . import utils

class Device(state.State):

    """ Basic device state. """

    def cmdline(self):
        """ Return a Linux cmdline parameter. """
        return self.get("cmdline")

class Driver(object):

    # The global set of all our device
    # classes. When we save / resume novm
    # state, we will look up drivers here
    # in order to run the appropriate fns.
    REGISTRY = {}

    @staticmethod
    def register(cls, driver=None):
        if driver is None:
            driver = cls.driver
        Driver.REGISTRY[driver] = cls

    @staticmethod
    def lookup(driver):
        return Driver.REGISTRY[driver]

    @property
    def name(self):
        """ Return a simple identifier for this device. """
        return str(uuid.uuid4())

    @property
    def debug(self):
        """ Return whether this device is debugging. """
        return False

    @property
    def driver(self):
        """ Return the driver for novmm. """
        raise NotImplementedError()

    def create(self,
            driver=None,
            name=None,
            data=None,
            debug=False,
            cmdline=None):

        """ Create a new device. """
        return Device(
            driver=driver or self.driver,
            name=name or self.name,
            data=data or {},
            debug=utils.asbool(debug) or self.debug,
            cmdline=cmdline or None)
