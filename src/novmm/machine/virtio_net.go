package machine

/*
#include <errno.h>
#include <sys/uio.h>
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
    int* sizes,
    int send) {

    int vecno;
    int rval;
    struct iovec vec[count+1];

    vec[0].iov_base = &pi_header;
    vec[0].iov_len = sizeof(pi_header);

    for (vecno = 0; vecno < count; vecno += 1) {
        vec[vecno+1].iov_base = (char*)ptrs[vecno];
        vec[vecno+1].iov_len = sizes[vecno];
    }

    if (send) {
        rval = writev(fd, &vec[0], count+1);
    } else {
        rval = readv(fd, &vec[0], count+1);
    }

    if (rval < 0) {
        return -errno;
    } else {
        return rval;
    }
}
*/
import "C"

import (
    "novmm/platform"
    "syscall"
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
    recv bool) error {

    ptrs := make([]unsafe.Pointer, 0, 0)
    sizes := make([]C.int, 0, 0)

    // Doing send or recv?
    var is_send C.int
    if recv {
        is_send = C.int(0)
    } else {
        is_send = C.int(1)
    }

    for bufs := range vchannel.incoming {

        // Legit?
        if len(bufs) < 1 || len(bufs[0].data) < 4 {
            vchannel.outgoing <- bufs
            continue
        }

        // Crop our header.
        if recv {
            header_len := 10
            bufs[0].data = bufs[0].data[header_len:]
        } else {
            header_len := int(bufs[0].data[2]) + int(bufs[0].data[3])<<8
            bufs[0].data = bufs[0].data[header_len:]
        }

        // Collect all our buffers.
        for _, buf := range bufs {
            if len(buf.data) > 0 {
                ptrs = append(ptrs, unsafe.Pointer(&buf.data[0]))
                sizes = append(sizes, C.int(len(buf.data)))
            }
        }

        // Send the constructed vector.
        rval := C.do_iovec(
            C.int(device.Fd),
            C.int(len(ptrs)),
            &ptrs[0],
            &sizes[0],
            is_send)
        if rval < C.int(0) {
            return syscall.Errno(int(-rval))
        }

        // Set the lengths written.
        for _, buf := range bufs {
            if len(buf.data) >= int(rval) {
                rval -= C.int(len(buf.data))
            } else {
                buf.data = buf.data[0:int(rval)]
                rval = C.int(0)
            }
        }

        // Return the buffers.
        vchannel.outgoing <- bufs

        // Reslice.
        ptrs = ptrs[0:0]
        sizes = sizes[0:0]
    }

    return nil
}

func NewVirtioMmioNet(info *DeviceInfo) (Device, error) {
    device, err := NewMmioVirtioDevice(info, VirtioTypeNet)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
    return &VirtioNetDevice{VirtioDevice: device}, err
}

func NewVirtioPciNet(info *DeviceInfo) (Device, error) {
    device, err := NewPciVirtioDevice(info, PciClassNetwork, VirtioTypeNet)
    device.Channels[0] = device.NewVirtioChannel(256)
    device.Channels[1] = device.NewVirtioChannel(256)
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
