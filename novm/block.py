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
Block device functions.
"""
import os

from . import virtio
from . import utils

class Disk(virtio.Driver):

    """ A Virtio block device. """

    virtio_driver = "block"

    def create(self,
            index=0,
            filename=None,
            dev=None,
            **kwargs):

        if filename is None:
            filename = "/dev/null"
        if dev is None:
            dev = "vd" + chr(ord("a") + index)

        # Open the device.
        f = open(filename, 'r+b')
        fd = os.dup(f.fileno())
        utils.clear_cloexec(fd)

        return super(Disk, self).create(data={
                "dev": dev,
                "fd": fd,
            }, **kwargs)

virtio.Driver.register(Disk)
