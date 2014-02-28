package machine

import (
    "novmm/platform"
)

//
// I/O events & operations --
//
// All I/O events (PIO & MMIO) are constrained to one
// simple interface. This simply makes writing devices
// a bit easier as you the read/write functions that
// must be implemented in each case are identical.
//
// This is a decision that could be revisited.

type IoEvent interface {
    Size() uint

    GetData() uint64
    SetData(val uint64)

    IsWrite() bool
}

type IoOperations interface {
    Read(offset uint64, size uint) (uint64, error)
    Write(offset uint64, size uint, value uint64) error
}

//
// I/O queues --
//
// I/O requests are serviced by a single go-routine,
// which pulls requests from a channel, performs the
// read/write as necessary and sends the result back
// on a requested channel.
//
// This structure was selected in order to allow all
// devices to operate without any locks and allowing
// their internal operation to be concurrent with the
// rest of the system.

type IoRequest struct {
    event  IoEvent
    offset uint64
    result chan error
}

type IoQueue chan IoRequest

func (queue IoQueue) Submit(event IoEvent, offset uint64) error {

    // Send the request to the device.
    req := IoRequest{event, offset, make(chan error)}
    queue <- req

    // Pull the result when it's done.
    return <-req.result
}

//
// I/O Handler --
//
// A handler represents a device instance, combined
// with a set of operations (typically for a single address).
// Effectively, this is the unit of concurrency and would
// represent a single port for a single device.

type IoHandler struct {
    Device

    start      platform.Paddr
    operations IoOperations
    queue      IoQueue
}

func NewIoHandler(
    device Device,
    start platform.Paddr,
    operations IoOperations) *IoHandler {

    io := &IoHandler{
        Device:     device,
        start:      start,
        operations: operations,
        queue:      make(IoQueue),
    }

    // Start the handler.
    go io.Run()

    return io
}

func normalize(val uint64, size uint) uint64 {
    switch size {
    case 1:
        return val & 0xff
    case 2:
        return val & 0xffff
    case 4:
        return val & 0xffffffff
    }
    return val
}

func (io *IoHandler) Run() {

    for {
        // Pull first request.
        req := <-io.queue
        size := req.event.Size()

        // Perform the operation.
        if req.event.IsWrite() {
            val := normalize(req.event.GetData(), size)
            err := io.operations.Write(req.offset, size, val)

            // Debug?
            io.Debug(
                "write %x @ %x [size: %d]",
                val,
                io.start.After(req.offset),
                size)

            req.result <- err

        } else {
            val, err := io.operations.Read(req.offset, size)
            val = normalize(val, size)
            if err == nil {
                req.event.SetData(val)
            }

            req.result <- err

            // Debug?
            io.Debug(
                "read %x @ %x [size: %d]",
                val,
                io.start.After(req.offset),
                size)
        }
    }
}
