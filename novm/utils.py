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
Miscellaneous functions.
"""
import os
import sys
import ctypes
import signal
import zipfile
import hashlib
import fcntl

def cleanup(fcn=None, *args, **kwargs):
    # We setup a safe procedure here to ensure that the
    # child will receive a SIGTERM when the parent exits.
    # This is used to cleanup all sorts of things once the
    # VM finishes automatically (like removing file trees,
    # removing tap devices, kill dnsmasq, etc.).
    libc = ctypes.CDLL("libc.so.6")

    if fcn is None:
        # This will only exit when the parent dies,
        # we will not run any function. This will be
        # used generally as the subprocess preexec_fn.
        child_pid = 0
        parent_pid = os.getppid()
    else:
        # Open a pipe to notify the parent when we're
        # ready to handle parent death and run the fcn.
        rpipe, wpipe = os.pipe()

        # Fork a child process.
        # This child process will execute the given code
        # when its parent dies. It's normally used inline.
        child_pid = os.fork()
        parent_pid = os.getppid()

        if child_pid == 0:
            os.close(rpipe)
        else:
            # Wait for the child to finish setup.
            # When it writes to its end of the pipe,
            # we know that it is prepared to handle
            # parent death and run the function.
            os.close(wpipe)
            os.read(rpipe, 1)
            os.close(rpipe)

    if child_pid == 0:
        # Set P_SETSIGDEATH to SIGTERM.
        libc.prctl(1, signal.SIGTERM)

        # Did we catch a race above, where we've
        # missed the re-parenting to init?
        if os.getppid() != parent_pid:
            os.kill(os.getpid(), signal.SIGTERM)

        # Are we finished?
        # In the case of not having a function to
        # execute, we simply return control. This is
        # a pre-exec hook for subprocess, for eaxample.
        if fcn is None:
            return

        # Close descriptors.
        # (Make sure we don't close our pipe).
        for fd in range(3, os.sysconf("SC_OPEN_MAX")):
            try:
                if fd != wpipe:
                    os.close(fd)
            except OSError:
                pass

        # Eat a signal.
        def squash(*args):
            pass

        # Suppress SIGINT. We'll get it when the user
        # hits Ctrl-C, which may be before our parent dies.
        signal.signal(signal.SIGINT, squash)

        # Temporarily supress SIGTERM, we do this until
        # it's re-enabled in the main wait loop below.
        signal.signal(signal.SIGTERM, squash)

        # Notify that we are ready to go.
        os.write(wpipe, b"o")
        os.close(wpipe)

        try:
            def interrupt(*args):
                sys.exit(0)

            # Get ready to receive our SIGTERM.
            # NOTE: When we receive the exception,
            # it will automatically be suppressed.
            signal.signal(signal.SIGTERM, interrupt)

            while True:
                # Catch our race condition.
                if os.getppid() != parent_pid:
                    break

                # Wait for a signal.
                signal.pause()
        except (SystemExit, KeyboardInterrupt):
            pass

        try:
            fcn(*args, **kwargs)
        except:
            # We eat all exceptions from the
            # cleanup function. If the user wants
            # to generate any output, they may --
            # however by default we silence it.
            pass
        os._exit(0)

def packdir(path, output, include=None, exclude=None):
    if include is None:
        include = ()
    if exclude is None:
        exclude = ()

    zipf = zipfile.ZipFile(output, 'w', allowZip64=True)

    for root, _, files in os.walk(path):
        for filename in files:

            # Check for exclusion.
            full_path = os.path.join(root, filename)
            in_exclude = False
            in_include = False
            for exclude_path in exclude:
                if exclude_path.startswith(full_path):
                    in_exclude = True
                    break
            for include_path in include:
                if include_path.startswith(full_path):
                    in_include = True
                    break
            if in_exclude or (len(include) > 0 and not in_include):
                continue

            zipf.write(full_path, os.path.relpath(full_path, path))

    return zipf

def unpackdir(path, output):
    zipf = zipfile.ZipFile(path, allowZip64=True)
    zipf.extractall(output)

def libexec(name):
    bindir = os.path.dirname(sys.argv[0])
    binname = os.path.basename(sys.argv[0])
    libexec_dir = os.path.join(bindir, "..", "lib", binname, "libexec")
    libexec_path = os.path.abspath(os.path.join(libexec_dir, name))
    if os.path.exists(libexec_path):
        return libexec_path
    else:
        alt_dir = os.path.join(bindir, "..", "bin")
        return os.path.abspath(os.path.join(alt_dir, name))

def asbool(value):
    if value is None:
        return False
    elif isinstance(value, bool):
        return value
    elif isinstance(value, str):
        return value.lower() == "true" or value.lower() == "yes"
    else:
        return False

def clear_cloexec(fd):
    flags = fcntl.fcntl(fd, fcntl.F_GETFD)
    fcntl.fcntl(fd, fcntl.F_SETFD, flags & (~fcntl.FD_CLOEXEC))

def copy(dst, src, sparse=False, hash=False):
    if hash:
        sha1 = hashlib.new('sha1')

    if sparse:
        # Grab our starting point.
        reloff = os.lseek(src.fileno(), 0, os.SEEK_CUR)

    while True:
        # Our read size.
        chunk = 65536

        if sparse:
            try:
                # Find the next chunk w/ SEEK_DATA.
                dataoffset = os.lseek(src.fileno(), reloff, 3)
            except OSError:
                # There is no more data.
                break
            skip = dataoffset - reloff
            if skip > 0:
                dst.seek(skip, os.SEEK_CUR)
                reloff += skip

            # Find the next hole w/ SEEK_HOLE.
            holeoffset = os.lseek(src.fileno(), 0, 4)

            # Return to the original location.
            os.lseek(src.fileno(), dataoffset, os.SEEK_SET)
            if holeoffset - dataoffset < 65536:
                chunk = holeoffset - dataoffset

        data = src.read(chunk)
        if not data:
            break
        dst.write(data)
        if sparse:
            reloff += len(data)

        if hash:
            sha1.update(data)

    dst.flush()
    if hash:
        return sha1.hexdigest()
