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
