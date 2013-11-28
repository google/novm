"""
Miscellaneous functions.
"""
import os
import sys
import ctypes
import signal
import thread
import zipfile

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
        parent_pid = os.getppid()
        child_pid = 0
    else:
        # Fork a child process.
        # This child process will execute the given code
        # when its parent dies. It's normally used inline.
        parent_pid = os.getpid()
        child_pid = os.fork()

    if child_pid == 0:
        # Cause a TERM to be handled as an exit.
        # We don't have multiple threads here, so we're
        # guaranteed that this will be handled in this frame.
        if fcn is not None:
            signal.signal(signal.SIGTERM, lambda signo, frame: thread.exit)

        # Set P_SETSIGDEATH to SIGTERM.
        libc.prctl(1, signal.SIGTERM)

        try:
            # Did we catch a race above, where we've
            # missing the re-parenting to init?
            if os.getppid() != parent_pid:
                os.kill(os.getpid(), signal.SIGTERM)

            # Are we finished?
            # In the case of not having a function to
            # execute, we simply return control.
            if fcn is None:
                return

            # Wait for the exit.
            while True:
                signal.pause()

        except SystemExit:
            if fcn is not None:
                fcn(*args, **kwargs)
            os._exit(0)

def zipdir(path, output, include=None, exclude=None):
    if include is None:
        include = ()
    if exclude is None:
        exclude = ()

    zipf = zipfile.ZipFile(output, 'w')

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

def libexec(name):
    bindir = os.path.dirname(sys.argv[0])
    binname = os.path.basename(sys.argv[0])
    libexec_dir = os.path.join(bindir, "..", "lib", binname, "libexec")
    return os.path.join(libexec_dir, name)

def asbool(value):
    if value is None:
        return False
    elif isinstance(value, bool):
        return value
    elif isinstance(value, str) or isinstance(value, unicode):
        return value.lower() == "true" or value.lower() == "yes"
    else:
        return False
