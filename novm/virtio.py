"""
Virtio device specification.
"""

from . import device

class Driver(device.Driver):

    @staticmethod
    def register(cls):
        device.Driver.register(cls, driver="virtio-pci-%s" % cls.virtio_driver)
        device.Driver.register(cls, driver="virtio-mmio-%s" % cls.virtio_driver)

    @property
    def virtio_driver(self):
        raise NotImplementedError()

    def create(self,
            index=-1,
            pci=False,
            data=None,
            **kwargs):

        if data is None:
            data = {}

        if pci:
            driver = "virtio-pci-%s" % self.virtio_driver
        else:
            driver = "virtio-mmio-%s" % self.virtio_driver

        if index >= 0 and not pci:
            # Are we an MMIO device?
            # NOTE: We arbitrarily pick 0xeXXXXXXX as the
            # start for all of our virtio devices. If we
            # have to do anymore reservation for I/O devices,
            # we might want to consider implemented something
            # a bit more thorough here.
            data["address"] = 0xe0000000 + index*4096
            data["interrupt"] = 32 + index
            cmdline = "virtio-mmio.%d@0x%x:%d:%d" % (
                index,
                0xe0000000 + index*4096,
                32 + index,
                index)
        else:
            cmdline = None

        return super(Driver, self).create(
            data=data,
            driver=driver,
            cmdline=cmdline,
            **kwargs)
