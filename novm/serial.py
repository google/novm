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
Console functions.
"""
from . import device
from . import virtio

class Console(virtio.Driver):

    """ A Virtio serial/console device. """

    virtio_driver = "console"

class Uart(device.Driver):

    driver = "uart"

    def com1(self, **kwargs):
        return self.create(data={
                "base": 0x3f8,
                "interrupt": 4,
            },
            cmdline="console=uart,io,0x3f8",
            **kwargs)

    def com2(self, **kwargs):
        return self.create(data={
                "base": 0x2f8,
                "interrupt": 3,
            },
            cmdline="console=uart,io,0x2f8",
            **kwargs)

virtio.Driver.register(Console)
device.Driver.register(Uart)
