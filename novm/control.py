"""
Control functions.
"""
import os
import socket
import select
import json
import binascii
import sys
import termios
import tty
import uuid

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

    def rpc(self, name, **kwargs):

        # Open the socket.
        fobj = self._sock.makefile(bufsize=0)
        fobj.write("NOVM RPC\n")

        # Write our command.
        # NOTE: This is currently not thread safe.
        # At some point, this class could implement
        # some multiplexing or safety for multiple
        # callers, but this isn't necessary until
        # there's a user other than the CLI.
        rpc_uuid = str(uuid.uuid4())
        rpc_cmd = {
            "method": "Control.%s" % name.title(),
            "params": [kwargs], 
            "id": rpc_uuid,
        }
        json.dump(rpc_cmd, fobj)
        fobj.write("\n")
        fobj.flush()

        # Get our result.
        obj = json.loads(fobj.readline())
        if obj["id"] != rpc_uuid:
            raise Exception(obj)
        if obj.get("error"):
            raise Exception(obj.get("error"))

        return obj.get("result")

    def run(self, command, env=None, cwd=None):
        if env is None:
            env = ["%s=%s" % (k,v) for (k,v) in os.environ.items()]
        if cwd is None:
            cwd = "/"

        fobj = self._sock.makefile(bufsize=0)
        fobj.write("NOVM RUN\n")

        # Write the initial run command.
        # This is actually a pass-through for the
        # guest Server.Start() command, although we
        # don't get to see the result.
        start_cmd = {
            "command": command,
            "environment": env,
            "cwd": cwd,
        }
        json.dump(start_cmd, fobj)
        fobj.flush()

        # Check for a basic error.
        obj = json.loads(fobj.readline())
        if obj is not None:
            raise Exception(obj)

        # Remember our exitcode.
        exitcode = 1

        try:
            # Save our terminal attributes and
            # enable raw mode for the terminal.
            orig_tc_attrs = termios.tcgetattr(0)
            tty.setraw(0)

            # Poll and transform the event stream.
            # This will basically turn this process
            # into a proxy for the remote process.
            read_set = [fobj, sys.stdin]

            while True:
                to_read, _, _ = select.select(read_set, [], [])

                if fobj in to_read:
                    # Decode the object.
                    obj = json.loads(fobj.readline())

                    if obj is None:
                        # Server is done.
                        raise IOError()

                    elif isinstance(obj, str) or isinstance(obj, unicode):
                        data = binascii.a2b_base64(obj)
                        sys.stdout.write(data)
                        sys.stdout.flush()

                    elif isinstance(obj, int):
                        # Remember our exitcode.
                        exitcode = obj

                if sys.stdin in to_read:

                    data = os.read(sys.stdin.fileno(), 4096)
                    if data:
                        data = binascii.b2a_base64(data)
                    json.dump(data, fobj)
                    fobj.write("\n")
                    fobj.flush()
                    if not data:
                        # We don't close the socket.
                        # We simply stop sending data.
                        read_set.remove(sys.stdin)

        except IOError:
            # Socket was closed.
            sys.exit(exitcode)

        finally:
            # Restore all of our original terminal attributes.
            termios.tcsetattr(0, termios.TCSAFLUSH, orig_tc_attrs)
