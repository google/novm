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

    def run(self, command, environment=None, cwd=None):
        if environment is None:
            environment = os.environ
        if cwd is None:
            cwd = "/"

        fobj = self._sock.makefile()
        fobj.write("NOVM RUN\n")

        # Write the initial run command.
        # This is actually a pass-through for the
        # guest Server.Start() command, although we
        # don't get to see the result.
        start_cmd = {
            "command": command,
            "environment": [
                "%s=%s" % (k,v) for (k,v) in environment.items()
            ],
            "cwd": cwd
        }
        json.dump(start_cmd, fobj)
        fobj.flush()

        # Check for a basic error.
        obj = json.loads(fobj.readline())
        if obj is not None:
            raise Exception(obj)

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
                        sys.stdout.close()

                    elif isinstance(obj, str) or isinstance(obj, unicode):
                        data = binascii.a2b_base64(obj)
                        sys.stdout.write(data)
                        sys.stdout.flush()

                    elif isinstance(obj, int):
                        sys.exit(obj)

                if sys.stdin in to_read:

                    data = os.read(sys.stdin.fileno(), 4096)
                    if data:
                        data = binascii.b2a_base64(data)
                    json.dump(data, fobj)
                    fobj.flush()
                    if not data:
                        # We don't close the socket.
                        # We simply stop sending data.
                        read_set.remove(sys.stdin)

        except IOError:
            # We don't expect the socket to be closed.
            # This is not a normal exit scenario (anymore).
            sys.exit(1)

        finally:
            # Restore all of our original terminal attributes.
            termios.tcsetattr(0, termios.TCSAFLUSH, orig_tc_attrs)
