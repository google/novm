"""
Memory devices.
"""
import os
import tempfile

from . import device

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
