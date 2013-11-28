"""
Filesystem device functions.
"""

from . import virtio

class FS(virtio.Device):

    def __init__(
            self,
            pack=None,
            read=None,
            write=None,
            **kwargs):

        super(FS, self).__init__(**kwargs)

        if read is None:
            read = []
        if write is None:
            write = []

        # Append our read mapping.
        read_info = {'/':[]}
        for path in read:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                read_info['/'].append(path)
            else:
                read_info[spec[1]].append(spec[0])
        for p in pack:
            path = self._packs.file(p)
            read_info['/'].append(path)

        args.append("-readfs=%s" % json.dumps(read_info))

        # Append our write mapping.
        tempdir = tempfile.mkdtemp()
        utils.cleanup(shutil.rmtree, tempdir)
        write_info = {'/': [tempdir]}

        for path in write:
            spec = path.split("=>", 1)
            if len(spec) == 1:
                write_info['/'].append(path)
            else:
                write_info[spec[1]].append(spec[0])

        args.append("-writefs=%s" % json.dumps(write_info))

        # Save our arguments.
        self._info = {
            "device": device,
            "filename": filename,
        }

        # Open the device.
        self._file = open(filename, 'w+b')

    def device(self):
        return super(Disk, self)._device(
            driver="fs",
            data={
                "device": self._info["device"],
                "fd": self._file.fileno(),
            }
        )

    def info(self):
        return self._info
