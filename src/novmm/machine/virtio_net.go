package machine

/*
#include <netinet/in.h>
#include <netinet/ip.h>
#include <net/if.h>
#include <linux/if_tun.h>
#include <linux/if_tunnel.h>

static struct tun_pi pi_header;

static inline void init_headers() {
    pi_header.flags = 0;
    pi_header.proto = __cpu_to_be16(ETH_P_IP);
}

static inline int do_iovec(
    int fd,
    int count,
    void** ptrs,
    int* szs,
    int send) {

    int vecno;
    struct msghdr hdr;
    struct iovec vec[count+1];

    hdr.msg_name = 0;
    hdr.msg_namelen = 0;
    hdr.msg_iov = &vec[0];
    hdr.msg_iovlen = count;
    hdr.msg_control = 0;
    hdr.msg_controllen = 0;
    hdr.msg_flags = 0;

    vec[0].iov_base = &pi_header;
    vec[0].iov_len = sizeof(pi_header);

    for (vecno = 0; vecno < count; vecno += 1) {
        vec[vecno+1].iov_base = (char*)ptrs[vecno];
        vec[vecno+1].iov_len = szs[vecno];
    }

    if (send) {
        return sendmsg(fd, &hdr, 0);
    } else {
        return recvmsg(fd, &hdr, 0);
    }
}
*/
import "C"

import (
    "novmm/platform"
    "unsafe"
)

type VirtioNetDevice struct {
    *VirtioDevice

    // The tap device file descriptor.
    Fd  int `json:"fd"`

    // The mac address.
    Mac string `json:"mac"`
}

func (device *VirtioNetDevice) processPackets(
    vchannel *VirtioChannel,
    recv bool) {

    ptrs := make([]unsafe.Pointer, 0, 0)
    szs := make([]C.int, 0, 0)

    // Doing send or recv?
    var is_send C.int
    if recv {
        is_send = C.int(0)
    } else {
        is_send = C.int(1)
    }

    for bufs := range vchannel.incoming {

        // Collect all our buffers.
        for _, buf := range bufs {
            ptrs = append(ptrs, unsafe.Pointer(&buf.data[0]))
            szs = append(szs, C.int(len(buf.data)))
        }

        // Send the constructed vector.
        C.do_iovec(
            C.int(device.Fd),
            C.int(len(bufs)),
            &ptrs[0],
            &szs[0],
            is_send)

        // Return the buffers.
        vchannel.outgoing <- bufs

        if recv {
            device.Debug("recv iovec w/ %d buffers...", len(bufs))
        } else {
            device.Debug("send iovec w/ %d buffers...", len(bufs))
        }

        // Reslice.
        ptrs = ptrs[0:0]
        szs = szs[0:0]
    }
}

func NewVirtioMmioNet(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeNet)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    device.Channels[2] = device.NewVirtioChannel(16)
    return &VirtioNetDevice{VirtioDevice: device}, err
}

func NewVirtioPciNet(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassNetwork, VirtioTypeNet)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    device.Channels[2] = device.NewVirtioChannel(16)
    return &VirtioNetDevice{VirtioDevice: device}, err
}

func (net *VirtioNetDevice) Attach(vm *platform.Vm, model *Model) error {
    err := net.VirtioDevice.Attach(vm, model)
    if err != nil {
        return err
    }

    // Start our network process.
    go net.processPackets(net.Channels[0], false)
    go net.processPackets(net.Channels[1], true)

    return nil
}

func init() {
    C.init_headers()
}
