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

    def __init__(self,
            path,
            bind=False,
            **kwargs):

        self._sent_rpc = False
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
        utils.clear_cloexec(self._sock.fileno())
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
        if not self._sent_rpc:
            self._sent_rpc = True
            fobj.write("NOVM RPC\n")
            fobj.flush()

        # Write our command.
        # NOTE: This is currently not thread safe.
        # At some point, this class could implement
        # some multiplexing or safety for multiple
        # callers, but this isn't necessary until
        # there's a user other than the CLI.
        rpc_uuid = str(uuid.uuid4())
        rpc_cmd = {
            "method": "Rpc.%s" % name.title(),
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

    def run(self, command, env=None, cwd=None, terminal=False):
        if env is None:
            env = ["%s=%s" % (k, v) for (k, v) in list(os.environ.items())]
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
            "terminal": terminal,
            "cwd": cwd,
        }
        json.dump(start_cmd, fobj)
        fobj.flush()

        try:
            try:
                # Save our terminal attributes and
                # enable raw mode for the terminal.
                if terminal:
                    orig_tc_attrs = termios.tcgetattr(0)
                    tty.setraw(0)
                    is_terminal = True
                else:
                    is_terminal = False
            except:
                is_terminal = False

            # Remember our exitcode.
            exitcode = 1

            # Expect our first object to be a None.
            # This indicates that we've started with
            # no errors. If it's a string -- error.
            started = False

            # We use a standard SSH-like escape.
            # Remember if we've seen an ~ already.
            seen_tilde = False

            # Poll and transform the event stream.
            # This will basically turn this process
            # into a proxy for the remote process.
            read_set = [fobj, sys.stdin]

            while True:
                to_read, _, _ = select.select(read_set, [], [])

                if fobj in to_read:
                    # Decode the object.
                    obj = json.loads(fobj.readline())

                    if not started:
                        # See note abouve started.
                        if obj is not None:
                            raise Exception("%s: %s" % (obj, command))
                        started = True
                        continue

                    if obj is None:
                        # Server is done.
                        raise IOError()

                    elif isinstance(obj, basestring):
                        data = binascii.a2b_base64(obj)
                        sys.stdout.write(data)
                        sys.stdout.flush()

                    elif isinstance(obj, int):
                        # Remember our exitcode.
                        exitcode = obj

                if sys.stdin in to_read:

                    data = os.read(sys.stdin.fileno(), 4096)

                    if is_terminal:
                        if data == "~":
                            seen_tilde = not seen_tilde
                        elif seen_tilde and data == ".":
                            break
                        elif seen_tilde:
                            seen_tilde = False

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
            if is_terminal:
                # Restore all of our original terminal attributes.
                termios.tcsetattr(0, termios.TCSAFLUSH, orig_tc_attrs)
