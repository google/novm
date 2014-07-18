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
Python adaptation of <asm-generic/ioctl.h>.
"""
import platform

# Ioctl numbers are encoded in 32 bits:
#   1  - 16: command
#   17 - 30: param struct size
#   31 - 32: direction
#
# Technically these offsets are architecture dependant, but in practice they don't
# change. In particular, on x86_32 and x86_64, they're the same.

# Argument sizes. Assumes python is same arch as kernel. That is, you can't make
# an ioctl from a 32-bit python process into a 64-bit kernel. Technically,
# there's nothing stopping you from doing so, but your structure sizes will be
# all messed up. We use platform.machine(), which is AMD64 or x86_64 on 64-bit,
# to identify kernel bit-ness.
#
# In addition to determining argument sizes, we also provide the correct format
# string to use for (unsigned) long in struct format strings. Unfortunately, in
# the struct module, sizes cannot be both native and unpadded. If you use an
# unpadded format specifier ('='), as you should in an ABI, then you always get
# the "standard" (unsigned) long size of 4, which makes no sense on a 64-bit
# kernel. To use use a 64-bit kernel's long size in a packed struct, you have to
# use the "standard" long long format. So (UNSIGNED_)LONG_PACKED_FMT is
# exported with the correct format.
if '64' in platform.machine():
    POINTER_SIZE = 8
    LONG_PACKED_FMT = 'q'
else:
    POINTER_SIZE = 4
    LONG_PACKED_FMT = 'l'
UNSIGNED_LONG_PACKED_FMT = LONG_PACKED_FMT.upper()
LONG_SIZE = UNSIGNED_LONG_SIZE = POINTER_SIZE
UNSIGNED_LONG_LONG_SIZE = 8

# Number of bits per field.
_IOC_NRBITS = 8
_IOC_TYPEBITS = 8
_IOC_SIZEBITS = 14
_IOC_DIRBITS = 2

# Shifts for each field.
_IOC_NRSHIFT = 0
_IOC_TYPESHIFT = _IOC_NRSHIFT + _IOC_NRBITS
_IOC_SIZESHIFT = _IOC_TYPESHIFT + _IOC_TYPEBITS
_IOC_DIRSHIFT = _IOC_SIZESHIFT + _IOC_SIZEBITS

# Direction bits.
_IOC_NONE = 0
_IOC_WRITE = 1
_IOC_READ = 2

def _IOC(dir, type, nr, size):
    '''Encodes dir, type, nr, and size into 32-bit ioctl number.'''
    return (dir  << _IOC_DIRSHIFT)  | \
           (type << _IOC_TYPESHIFT) | \
           (nr   << _IOC_NRSHIFT)   | \
           (size << _IOC_SIZESHIFT)

_IOWR = lambda type, nr, size: _IOC(_IOC_READ | _IOC_WRITE, type, nr, size)
_IOW  = lambda type, nr, size: _IOC(_IOC_WRITE, type, nr, size)
_IOR  = lambda type, nr, size: _IOC(_IOC_READ, type, nr, size)
_IO   = lambda type, nr:       _IOC(_IOC_NONE, type, nr, 0)
