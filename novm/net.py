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
Network functions.
"""
import os
import fcntl
import struct
import random
import subprocess

from . import utils
from . import virtio
from . import ioctl

# Tap device flags.
IFF_TAP      = 0x0002
IFF_NO_PI    = 0x1000
IFF_VNET_HDR = 0x4000

# Tap device offloads.
TUN_F_CSUM    = 0x01
TUN_F_TSO4    = 0x02
TUN_F_TSO6    = 0x04
TUN_F_TSO_ECN = 0x08
TUN_F_UFO     = 0x10

# Tap device ioctls.
TUNSETIFF       = ioctl._IOW(ord('T'), 202, 4)
TUNGETFEATURES  = ioctl._IOR(ord('T'), 207, 4)
TUNSETOFFLOAD   = ioctl._IOW(ord('T'), 208, 4)
TUNGETVNETHDRSZ = ioctl._IOR(ord('T'), 215, 4)

def random_mac(oui="28:48:46"):
    """ Return a random MAC address. """
    suffix = ":".join(
        ["%02x" % random.randint(1, 254)
         for _ in range(3)])
    return "%s:%s" % (oui, suffix)

def parse_ipv4mask(ip):
    """
    Parse an IP address given CIDR form.

    We also return the associated gateway
    and broadcast address for the subnet.
    """
    (address, mask) = ip.split("/", 1)
    # Compute the relevant masks.
    parts = [int(part) for part in address.split(".")]
    addr = sum([parts[i]<<(24-i*8) for i in range(len(parts))])
    mask = ((1<<int(mask))-1) << (32-int(mask))

    # Compute the addresses.
    network_addr = addr & mask
    first_addr = network_addr + 1
    broadcast_addr = network_addr + (~mask & (mask-1))
    end_addr = broadcast_addr - 1

    def st(addr):
        return ".".join([
            str((addr>>24) & 0xff),
            str((addr>>16) & 0xff),
            str((addr>>8) & 0xff),
            str((addr>>0) & 0xff),
        ])

    return address, st(first_addr), st(end_addr)

def tap_device(name):
    """ Create a tap device. """
    tap = open('/dev/net/tun', 'r+b')

    # Figure out if the kernel supports processing vnet headers on tap
    # devices. This is necessary for forwarding hardware offloading
    # from the guest virtual nics to the physical nics on the host.
    features_raw = fcntl.ioctl(tap, TUNGETFEATURES, struct.pack('I', 0))
    features = struct.unpack('I', features_raw)[0]

    flags = IFF_TAP | IFF_NO_PI
    if features & IFF_VNET_HDR:
        flags |= IFF_VNET_HDR
        strip_vnet_hdr = False
    else:
        strip_vnet_hdr = True

    # Create the tap device.
    ifr = struct.pack('16sH', name, flags)
    fcntl.ioctl(tap, TUNSETIFF, ifr)

    vnet_hdr_sz_raw = fcntl.ioctl(tap, TUNGETVNETHDRSZ, struct.pack('I', 0))
    vnet_hdr_sz = struct.unpack('i', vnet_hdr_sz_raw)[0]

    # Size of the vnet header expected by the tap device.
    vnet = 0 if strip_vnet_hdr else vnet_hdr_sz

    # Enable hardware offloads.
    if vnet:
        try:
            fcntl.ioctl(tap, TUNSETOFFLOAD,
                        TUN_F_CSUM | TUN_F_TSO4 |
                        TUN_F_TSO6 | TUN_F_TSO_ECN | TUN_F_UFO)
            offload = True
        except Exception as ex:
            print("Failed to enable offloads:", ex)
            offload = False
    else:
        # Can't support offloads without vnet header.
        offload = False

    return tap, vnet, offload

class Nic(virtio.Driver):

    """ A Virtio network device. """

    virtio_driver = "net"

    def create(self,
            index=None,
            mac=None,
            tapname=None,
            bridge=None,
            ip=None,
            gateway=None,
            mtu=None,
            **kwargs):

        if mac is None:
            mac = random_mac()
        if tapname is None:
            tapname = "novm%d-%d" % (os.getpid(), index)

        # Create our new tap device.
        tap, vnet, offload = tap_device(tapname)
        if mtu is not None:
            subprocess.check_call(
                ["/sbin/ip", "link", "set", "dev", tapname, "mtu", str(mtu)],
                close_fds=True)

        # Enslave to the given bridge.
        # (It will automatically be removed.)
        if bridge is not None:
            subprocess.check_call(
                ["/sbin/brctl", "addif", bridge, tapname],
                close_fds=True)

        # Make sure the interface is up.
        subprocess.check_call(
            ["/sbin/ip", "link", "set", "up", "dev", tapname],
            close_fds=True)

        # Start our dnsmasq.
        # This is just a simple responded for
        # DHCP queries and routes that will die
        # whenever the underlying instance dies.
        if ip is not None:
            (address, start, end) = parse_ipv4mask(ip)
            dnsmasq_opts = ["/usr/sbin/dnsmasq"]
            dnsmasq_opts.append("--keep-in-foreground")
            dnsmasq_opts.append("--no-daemon")
            dnsmasq_opts.append("--conf-file=")
            dnsmasq_opts.append("--bind-interfaces")
            dnsmasq_opts.append("--except-interface=lo")
            dnsmasq_opts.append("--interface=%s" % tapname)
            dnsmasq_opts.append("--dhcp-range=%s,%s" % (start, end))
            dnsmasq_opts.append("--dhcp-host=%s,%s,%s" %
                (mac, address, kwargs.get("name")))
            if gateway is not None:
                dnsmasq_opts.append("--dhcp-option=option:router,%s" % gateway)

            # Run dnsmasq.
            subprocess.Popen(
                dnsmasq_opts,
                preexec_fn=utils.cleanup,
                close_fds=True,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE)
            ip = address
        else:
            ip = None

        fd = os.dup(tap.fileno())
        utils.clear_cloexec(fd)

        return super(Nic, self).create(data={
                "mac": mac,
                "fd": os.dup(tap.fileno()),
                "vnet": vnet,
                "fd": fd,
                "offload": offload,
                "ip": ip
            }, **kwargs)

virtio.Driver.register(Nic)
