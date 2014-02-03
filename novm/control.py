"""
Console functions.
"""
import os
import socket
import shutil
import select
import json
import binascii
import sys

from . import utils
from . import device
from . import virtio

class Control(object):

    def __init__(
            self,
            pid,
            bind=False,
            **kwargs):

        super(Control, self).__init__(**kwargs)

        try:
            os.makedirs("/var/run/novm")
        except OSError:
            # Exists.
            pass

        path = "/var/run/novm/%s.ctrl" % pid
        self._sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

        if bind:
            if os.path.exists(path):
                os.remove(path)
            utils.cleanup(os.remove, path)
            self._sock.bind(path)
            self._sock.listen(1024)
        else:
            self._sock.connect(path)

    def fd(self):
        return self._sock.fileno()

    def run(self, command, environment=None, cwd="/"):
        if environment is None:
            environment = os.environ

        fobj = self._sock.makefile()
        fobj.write("NOVM RUN\n")

        # Write the initial run command.
        json.dump({
            "command": command,
            "environment": [
                "%s=%s" % (k,v) for (k,v) in environment.items()
            ],
            "cwd": cwd
        }, fobj)
        fobj.flush()

        # Read the initial result.
        res = json.loads(fobj.readline())
        if not "pid" in res:
            raise Exception(res)

        try:
            # Poll and transform the event stream.
            # This will basically turn this process
            # into a proxy for the remote process.
            while True:
                to_read, _, _ = select.select([fobj, sys.stdin], [], [])

                if fobj in to_read:
                    obj = json.loads(fobj.readline())

                    if "stderr" in obj:
                        if obj["data"] is None:
                            if obj.get("stderr"):
                                sys.stderr.close()
                            else:
                                sys.stdout.close()
                        else:
                            data = binascii.a2b_base64(obj["data"])
                            if obj.get("stderr"):
                                sys.stderr.write(data)
                            else:
                                sys.stdout.write(data)

                    elif "exitcode" in obj:
                        sys.exit(obj["exitcode"])

                if sys.stdin in to_read:
                    data = os.read(sys.stdin.fileno(), 4096)
                    fobj.write(data)
                    fobj.flush()

        except IOError:
            # Socket eventually will be closed.
            # This is a clean exit scenario.
            sys.exit(0)
