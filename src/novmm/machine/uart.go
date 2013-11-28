package machine

import (
    "os"
    "sync"
)

const (
    UartDefaultRclk = 1843200
    UartDefaultBaud = 9600
)

const (
    UartFcrRXMASK = 0xc0
)

const (
    UartIerERXRDY = 0x1
    UartIerETXRDY = 0x2
    UartIerERLS   = 0x4
    UartIerEMSC   = 0x8
)

const (
    UartIirIMASK  = 0xf
    UartIirRXTOUT = 0xc
    UartIirBUSY   = 0x7
    UartIirRLS    = 0x6
    UartIirMLSC   = 0x5
    UartIirRXRDY  = 0x4
    UartIirTXRDY  = 0x2
    UartIirNOPEND = 0x1
)

const (
    UartLcrDLAB = 0x80
)

const (
    UartMsrPRESCALE = 0x80
    UartMsrLOOPBACK = 0x10
    UartMsrIE       = 0x08
    UartMsrDRS      = 0x04
    UartMsrRTS      = 0x02
    uartMsrDTR      = 0x01
    UartMsrMASK     = 0xf
)

const (
    UartLsrFIFO  = 0x80
    UartLsrTEMT  = 0x40
    UartLsrTHRE  = 0x20
    UartLsrBI    = 0x10
    UartLsrFE    = 0x08
    UartLsrPE    = 0x04
    UartLsrOE    = 0x02
    UartLsrRXRDY = 0x01
)

const (
    UartMsrDCD  = 0x80
    UartMsrRI   = 0x40
    UartMsrDSR  = 0x20
    UartMsrCTS  = 0x10
    UartMsrDDCD = 0x08
    UartMsrTERI = 0x04
    UartMsrDDSR = 0x02
    UartMsrDCTS = 0x01
)

type UartData struct {
    uart *Uart
}

type UartIntr struct {
    uart *Uart
}

func (uart *UartData) Read(offset uint64, size uint) (uint64, error) {
    if uart.uart.lcr.value&UartLcrDLAB != 0 {
        return uart.uart.dll.Read(offset, size)
    }

    uart.uart.Mutex.Lock()
    uart.uart.in_buffer -= 1
    if uart.uart.in_buffer == 0 {
        uart.uart.lsr.value = uart.uart.lsr.value & ^uint64(UartLsrRXRDY)
    }
    uart.uart.Mutex.Unlock()
    return uint64(<-uart.uart.buffer), nil
}

func (uart *UartData) Write(offset uint64, size uint, value uint64) error {
    if uart.uart.lcr.value&UartLcrDLAB != 0 {
        return uart.uart.dll.Write(offset, size, value)
    }

    // Ignore return value.
    os.Stdout.Write([]byte{byte(value)})
    return nil
}

func (uart *UartIntr) Read(offset uint64, size uint) (uint64, error) {
    if uart.uart.lcr.value&UartLcrDLAB != 0 {
        return uart.uart.dlh.Read(offset, size)
    }

    return uart.uart.ier.Read(offset, size)
}

func (uart *UartIntr) Write(offset uint64, size uint, value uint64) error {
    if uart.uart.lcr.value&UartLcrDLAB != 0 {
        return uart.uart.dlh.Write(offset, size, value)
    }

    return uart.uart.ier.Write(offset, size, value)
}

type Uart struct {
    ier Register
    iir Register
    lcr Register
    mcr Register
    lsr Register
    msr Register
    fcr Register
    scr Register

    dll Register
    dlh Register

    // Our Fifo.
    buffer    chan byte
    in_buffer int32
    sync.Mutex

    // Pending?
    thre_pending bool

    Interrupt int    `json:"interrupt"`
    IoBase    uint64 `json:"address"`
}

func (uart *Uart) readStream(input *os.File) error {

    buffer := make([]byte, 1, 1)

    for {
        n, err := input.Read(buffer)
        if n > 0 {
            uart.Mutex.Lock()
            uart.in_buffer += 1
            uart.lsr.value = uart.lsr.value | UartLsrRXRDY
            uart.Mutex.Unlock()
            uart.buffer <- buffer[0]
        }
        if err != nil {
            return err
        }
    }

    return nil
}

func NewUart(info *DeviceInfo) (*Uart, *Device, error) {

    // Create the uart.
    uart := new(Uart)
    uart.buffer = make(chan byte, 1024)
    info.Load(uart)

    // Is this a sane uart?
    if uart.IoBase == 0 || uart.Interrupt == 0 {
        return nil, nil, UartUnknown
    }

    // Start reading stdin.
    go uart.readStream(os.Stdin)

    // Create the device.
    device, err := NewDevice(
        info,
        IoMap{
            // Our configuration ports.
            MemoryRegion{0, 1}: &UartData{uart},
            MemoryRegion{1, 1}: &UartIntr{uart},
            MemoryRegion{2, 1}: &uart.iir, // Interrupt identification.
            MemoryRegion{3, 1}: &uart.lcr, // Line control register.
            MemoryRegion{4, 1}: &uart.mcr, // Modem control register.
            MemoryRegion{5, 1}: &uart.lsr, // Line status register.
            MemoryRegion{6, 1}: &uart.msr, // Modem status register.
            MemoryRegion{7, 1}: &uart.scr, // Scratch register.

            MemoryRegion{8, 2}:  &uart.dll, // Divisor low-register.
            MemoryRegion{10, 2}: &uart.dlh, // Divisor high-register.
        },
        uart.IoBase, // Port-I/O offset.
        IoMap{},
        0,  // Memory-I/O offset.
    )
    if err != nil {
        return nil, nil, err
    }

    // Set our readonly bits.
    uart.lsr.readonly = 0xff
    uart.msr.readonly = 0xff
    uart.msr.readclr = 0x0f
    uart.mcr.readonly = 0x1f

    // We're always ready for data.
    uart.lsr.value = UartLsrTEMT | UartLsrTHRE
    uart.lsr.readonly = uart.lsr.value

    // Clear the OE bit on read.
    uart.lsr.readclr = UartLsrOE

    // Set our divisor.
    divisor := uint64(UartDefaultRclk / UartDefaultBaud / 16)
    uart.dll.value = divisor
    uart.dlh.value = divisor >> 16

    return uart, device, nil
}

func (uart *Uart) getInterruptStatus() uint8 {
    uart.Mutex.Lock()
    defer uart.Mutex.Unlock()

    if uart.lsr.value&UartLsrOE != 0 && uart.ier.value&UartIerERLS != 0 {
        return UartIirRLS
    } else if uart.in_buffer > 0 && uart.ier.value&UartIerERXRDY != 0 {
        return UartIirRXTOUT
    } else if uart.thre_pending && uart.ier.value&UartIerETXRDY != 0 {
        return UartIirTXRDY
    } else if uart.msr.value&UartMsrMASK != 0 && uart.ier.value&UartIerEMSC != 0 {
        return UartIirMLSC
    }

    return UartIirNOPEND
}

func LoadUart(model *Model, info *DeviceInfo) error {

    _, device, err := NewUart(info)
    if err != nil {
        return err
    }

    return model.AddDevice(device)
}
