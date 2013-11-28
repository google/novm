package machine

import (
    "math"
    "time"
)

const (
    RtcSecond      = 0x00
    RtcSecondAlarm = 0x01
    RtcMinute      = 0x02
    RtcMinuteAlarm = 0x03
    RtcHour        = 0x04
    RtcHourAlarm   = 0x05
    RtcWeekday     = 0x06
    RtcDay         = 0x07
    RtcMonth       = 0x08
    RtcYear        = 0x09
    RtcStatusA     = 0xa
    RtcStatusB     = 0xb
    RtcIntr        = 0xc
    RtcStatusD     = 0xd
    RtcCentury     = 0x32
)

const (
    RtcStatusATUP = 0x80
)

const (
    RtcStatusBDST   = 0x01
    RtcStatusB24HR  = 0x02
    RtcStatusBBIN   = 0x04
    RtcStatusBPINTR = 0x40
    RtcStatusBHALT  = 0x80
)

const (
    RtcStatusDPWR = 0x80
)

//
// Rtc --
//
// A basic real-time clock. This simulates ticks via
// the system time (whenever it is read, we tick the delta).
//

type Rtc struct {
    last time.Time
    now  time.Time

    addr uint8

    alarmSecond uint8
    alarmMinute uint8
    alarmHour   uint8

    statusA uint8
    statusB uint8
}

type RtcAddr struct {
    *Rtc
}

type RtcData struct {
    *Rtc
}

func (rtc *Rtc) Update(alive bool) {

    wall_clock := time.Now()

    if alive {
        rtc.now = rtc.now.Add(wall_clock.Sub(rtc.last))
    }

    rtc.last = wall_clock
}

var Bin2Bcd = []uint8{
    0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09,
    0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19,
    0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29,
    0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39,
    0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49,
    0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59,
    0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69,
    0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79,
    0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89,
    0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99,
}

func (rtc *Rtc) Val(data int) uint64 {
    if rtc.statusB&RtcStatusBBIN != 0 {
        return uint64(uint8(data))
    }

    return uint64(Bin2Bcd[data%100])
}

func (reg *RtcAddr) Read(offset uint64, size uint) (uint64, error) {
    return uint64(reg.Rtc.addr), nil
}

func (reg *RtcAddr) Write(offset uint64, size uint, value uint64) error {
    reg.Rtc.addr = uint8(value)
    return nil
}

func (reg *RtcData) Read(offset uint64, size uint) (uint64, error) {

    // Tick.
    reg.Rtc.Update(reg.Rtc.statusB&RtcStatusBHALT != 0)

    switch reg.Rtc.addr {
    case RtcSecondAlarm:
        return uint64(reg.Rtc.alarmSecond), nil

    case RtcMinuteAlarm:
        return uint64(reg.Rtc.alarmMinute), nil

    case RtcHourAlarm:
        return uint64(reg.Rtc.alarmHour), nil

    case RtcSecond:
        return reg.Rtc.Val(reg.Rtc.now.Second()), nil

    case RtcMinute:
        return reg.Rtc.Val(reg.Rtc.now.Minute()), nil

    case RtcHour:
        if reg.Rtc.statusB&RtcStatusB24HR != 0 {
            return reg.Rtc.Val(reg.Rtc.now.Hour()), nil
        }

        // Top bit must be set in 12-hour format.
        // This is such a frustrating way to represent time.
        hour := reg.Rtc.Val(reg.Rtc.now.Hour() % 12)
        if reg.Rtc.now.Hour() >= 12 {
            return 0x80 | hour, nil
        }
        return hour, nil

    case RtcWeekday:
        return reg.Rtc.Val(int(reg.now.Weekday())), nil

    case RtcDay:
        return reg.Rtc.Val(reg.now.Day()), nil

    case RtcMonth:
        return reg.Rtc.Val(int(reg.now.Month())), nil

    case RtcYear:
        return reg.Rtc.Val(reg.now.Year()), nil

    case RtcStatusA:
        return uint64(reg.Rtc.statusA), nil

    case RtcStatusB:
        return uint64(reg.Rtc.statusB), nil

    case RtcIntr:
        return 0, nil

    case RtcStatusD:
        return RtcStatusDPWR, nil
    }

    return uint64(math.MaxUint64), nil
}

func (reg *RtcData) Write(offset uint64, size uint, value uint64) error {

    val := uint8(value)

    switch reg.Rtc.addr {

    case RtcStatusA:
        reg.Rtc.statusA = val & ^uint8(RtcStatusATUP)
        break

    case RtcStatusB:
        reg.Rtc.statusB = val
        break

    case RtcIntr:
        // Ignore.
        break

    case RtcStatusD:
        // Ignore.
        break

    case RtcSecondAlarm:
        reg.Rtc.alarmSecond = val
        break

    case RtcMinuteAlarm:
        reg.Rtc.alarmMinute = val
        break

    case RtcHourAlarm:
        reg.Rtc.alarmHour = val
        break
    }

    return nil
}

func NewRtc(info *DeviceInfo) (*Rtc, *Device, error) {

    // Create the rtc.
    rtc := new(Rtc)
    rtc.Update(true)
    info.Load(rtc)

    // Create the device.
    device, err := NewDevice(
        info,
        IoMap{
            // Our configuration ports.
            MemoryRegion{0x70, 1}: &RtcAddr{rtc},
            MemoryRegion{0x71, 1}: &RtcData{rtc},
        },
        0,  // Port-I/O offset.
        IoMap{},
        0,  // Memory-I/O offset.
    )

    // Return our bus and device.
    return rtc, device, err
}

func LoadRtc(model *Model, info *DeviceInfo) error {

    _, device, err := NewRtc(info)
    if err != nil {
        return err
    }

    return model.AddDevice(device)
}
