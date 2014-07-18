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
PCI functionality.
"""
from . import device

class PciBus(device.Driver):

    driver = "pci-bus"

    def create(self, **kwargs):
        return super(PciBus, self).create(
            cmdline="pci=conf1",
            **kwargs)

class PciHostBridge(device.Driver):

    # NOTE: For now, PCI support is pretty sketchy.
    # Generally, we'll need to have a hostbridge appear
    # in the list of devices.
    # For information on the bridge that might normally
    # appear, see src/novmm/machine/pcihost.go.

    driver = "pci-hostbridge"

device.Driver.register(PciBus)
device.Driver.register(PciHostBridge)
