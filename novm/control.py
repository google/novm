"""
Control functions.
"""
import os
import socket
import select
import json
import binascii
import sys

from . import utils

class Control(object):

    def __init__(
            self,
            path,
            bind=False,
            **kwargs):

        super(Control, self).__init__(**kwargs)

        dirname = os.path.dirname(path)
        if not os.path.exists(dirname) or not os.path.isdir(dirname):
            try:
                os.makedirs(dirname)
            except OSError:
                # Did we catch a race condition?
                if not os.path.exists(dirname) or not os.path.isdir(dirname):
                    raise

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

        # Our process exitcode.
        exitcode = 0

        try:
            # Poll and transform the event stream.
            # This will basically turn this process
            # into a proxy for the remote process.
            read_set = [fobj, sys.stdin]
            while True:
                to_read, _, _ = select.select(read_set, [], [])

                if fobj in to_read:
                    data = fobj.readline()

                    # Server has closed the socket?
                    if not data:
                        break

                    # Decode the object.
                    obj = json.loads(data)

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
                        exitcode = obj["exitcode"]

                if sys.stdin in to_read:

                    data = os.read(sys.stdin.fileno(), 4096)
                    if data:
                        fobj.write(data)
                        fobj.flush()
                    else:
                        # We don't close the socket.
                        read_set.remove(sys.stdin)

        except IOError:
            # Socket eventually will be closed.
            # This is a clean exit scenario.
            pass

        sys.exit(exitcode)
