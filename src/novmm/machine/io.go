package machine

import (
    "log"
    "math"
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
    MemoryRegion

    info       *DeviceInfo
    operations IoOperations
    queue      IoQueue
}

func NewIoHandler(
    info *DeviceInfo,
    region MemoryRegion,
    operations IoOperations) *IoHandler {

    io := &IoHandler{
        MemoryRegion: region,
        info:         info,
        operations:   operations,
        queue:        make(IoQueue),
    }

    // Start the handler.
    go io.Run()

    return io
}

type Register struct {
    value uint64

    // Read-only bits?
    readonly uint64

    // Clear these bits on read.
    readclr uint64
}

func (register *Register) Read(offset uint64, size uint) (uint64, error) {
    var mask uint64

    switch size {
    case 1:
        mask = 0x000000ff
    case 2:
        mask = 0x0000ffff
    case 3:
        mask = 0x00ffffff
    case 4:
        mask = 0xffffffff
    }

    value := uint64(math.MaxUint64)

    switch offset {
    case 0:
        value = (register.value) & mask
    case 1:
        value = (register.value >> 8) & mask
        mask = mask << 8
    case 2:
        value = (register.value >> 16) & mask
        mask = mask << 16
    case 3:
        value = (register.value >> 24) & mask
        mask = mask << 24
    }

    register.value = register.value & ^(mask & register.readclr)
    return value, nil
}

func (register *Register) Write(offset uint64, size uint, value uint64) error {
    var mask uint64

    switch size {
    case 1:
        mask = 0x000000ff & register.readonly
    case 2:
        mask = 0x0000ffff & register.readonly
    case 3:
        mask = 0x00ffffff & register.readonly
    case 4:
        mask = 0xffffffff & register.readonly
    }

    value = value & mask

    switch offset {
    case 1:
        mask = mask << 8
        value = value << 8
    case 2:
        mask = mask << 16
        value = value << 16
    case 3:
        mask = mask << 24
        value = value << 24
    }

    register.value = (register.value & ^mask) | (value & mask)
    return nil
}

func (io *IoHandler) Run() {

    for {
        // Pull first request.
        req := <-io.queue
        size := req.event.Size()

        // Limit the size.
        if req.offset+uint64(size) >= io.MemoryRegion.Size {
            size = uint(io.MemoryRegion.Size - req.offset)
        }

        // Perform the operation.
        if req.event.IsWrite() {
            val := req.event.GetData()
            err := io.operations.Write(
                req.offset,
                req.event.Size(),
                val)

            req.result <- err

            // Debug?
            if io.info.Debug {
                log.Printf("%s: write %x @ %x+%x",
                    io.info.Name,
                    val,
                    io.MemoryRegion.Start,
                    req.offset)
            }

        } else {
            val, err := io.operations.Read(
                req.offset,
                req.event.Size())
            if err == nil {
                req.event.SetData(val)
            }

            req.result <- err

            // Debug?
            if io.info.Debug {
                log.Printf("%s: read %x @ %x+%x",
                    io.info.Name,
                    val,
                    io.MemoryRegion.Start,
                    req.offset)
            }
        }
    }
}
