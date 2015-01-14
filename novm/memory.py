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
Memory devices.
"""
import os
import tempfile

from . import device
from . import utils

class UserMemory(device.Driver):

    driver = "user-memory"

    def create(self,
            size=None,
            fd=None,
            **kwargs):

        # No file given?
        if fd is None:
            with tempfile.NamedTemporaryFile() as tf:
                fd = os.dup(tf.fileno())
                utils.clear_cloexec(fd)

        # No size given? Default to file size.
        if size is None:
            fd_stat = os.fstat(fd)
            size = fd_stat.st_size

        # Truncate the file.
        os.ftruncate(fd, size)

        return super(UserMemory, self).create(data={
            "fd": fd,
            "size": size,
        }, **kwargs)

    def save(self, state, pid):
        """ Open up the fd and return it back. """
        return ({
            # Save the size of the memory block.
            "size": state.get("size"),
        }, {
            # Serialize the entire open fd.
            "memory": open("/proc/%d/fd/%d" % (pid, state["fd"]), "r")
        })

    def load(self, state, files):
        return self.create(
            size=state.get("size"),
            fd=files["memory"].fileno())

device.Driver.register(UserMemory)
