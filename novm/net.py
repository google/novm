"""
Network functions.
"""
import os
import fcntl
import struct
import random
import subprocess

from . import utils
from . import device
from . import virtio

def random_mac(oui="28:48:46"):
    """ Return a random MAC address. """
    suffix = ":".join(
        ["%x" % random.randint(1, 254)
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
    # Flags are IFF_TAP|IFF_NO_PI.
    ifr = struct.pack('16sH', name, 0x1002)
    fcntl.ioctl(tap, 0x400454ca, ifr)
    return tap

class Nic(virtio.Device):

    """ A Virtio network device. """

    virtio_driver = "net"

    def __init__(
            self,
            index=None,
            mac=None,
            tapname=None,
            bridge=None,
            ip=None,
            gateway=None,
            mtu=None,
            **kwargs):

        super(Nic, self).__init__(index=index, **kwargs)

        if mac is None:
            mac = random_mac()
        if tapname is None:
            tapname = "novm%d-%d" % (os.getpid(), index)

        # Save our arguments.
        self._info = {
            "mac": mac,
            "tapname": tapname,
            "bridge": bridge,
            "ip": ip,
            "gateway": gateway,
            "mtu": mtu,
        }

        # Create our new tap device.
        self._tap = tap_device(tapname)
        if mtu is not None:
            subprocess.check_call(
                ["ip", "link", "set", "dev", tapname, "mtu", str(mtu)],
                close_fds=True)

        # Enslave to the given bridge.
        # (It will automatically be removed.)
        if bridge is not None:
            subprocess.call(
                ["brctl", "addif", bridge, tapname],
                close_fds=True)

        # Make sure the interface is up.
        subprocess.check_call(
            ["ip", "link", "set", "up", "dev", tapname])

        # Start our dnsmasq.
        # This is just a simple responded for
        # DHCP queries and routes that will die
        # whenever the underlying instance dies.
        if ip is not None:
            (address, start, end) = parse_ipv4mask(ip)
            dnsmasq_opts = ["dnsmasq"]
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
                close_fds=True)
            self._ip = address
        else:
            self._ip = None

    def data(self):
        return {
            "mac": self._info["mac"],
            "fd": self._tap.fileno(),
        }

    def ip(self):
        return self._ip

    def info(self):
        return self._info
