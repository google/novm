// Copyright 2014 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package machine

import (
	"math"
	"novmm/platform"
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
	PioDevice

	// The time.
	Now  time.Time `json:"now"`
	last time.Time

	// Registers.
	Addr        uint8 `json:"selector"`
	AlarmSecond uint8 `json:"alarm-second"`
	AlarmMinute uint8 `json:"alarm-minute"`
	AlarmHour   uint8 `json:"alarm-hour"`
	StatusA     uint8 `json:"statusa"`
	StatusB     uint8 `json:"statusb"`
}

type RtcAddr struct {
	*Rtc
}

type RtcData struct {
	*Rtc
}

func (rtc *Rtc) Tick(alive bool) {

	wall_clock := time.Now()

	if alive {
		rtc.Now = rtc.Now.Add(wall_clock.Sub(rtc.last))
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
	if rtc.StatusB&RtcStatusBBIN != 0 {
		return uint64(uint8(data))
	}

	return uint64(Bin2Bcd[data%100])
}

func (reg *RtcAddr) Read(offset uint64, size uint) (uint64, error) {
	return uint64(reg.Rtc.Addr), nil
}

func (reg *RtcAddr) Write(offset uint64, size uint, value uint64) error {
	reg.Rtc.Addr = uint8(value)
	return nil
}

func (reg *RtcData) Read(offset uint64, size uint) (uint64, error) {

	reg.Rtc.Tick(reg.Rtc.StatusB&RtcStatusBHALT != 0)

	switch reg.Rtc.Addr {
	case RtcSecondAlarm:
		return uint64(reg.Rtc.AlarmSecond), nil

	case RtcMinuteAlarm:
		return uint64(reg.Rtc.AlarmMinute), nil

	case RtcHourAlarm:
		return uint64(reg.Rtc.AlarmHour), nil

	case RtcSecond:
		return reg.Rtc.Val(reg.Rtc.Now.Second()), nil

	case RtcMinute:
		return reg.Rtc.Val(reg.Rtc.Now.Minute()), nil

	case RtcHour:
		if reg.Rtc.StatusB&RtcStatusB24HR != 0 {
			return reg.Rtc.Val(reg.Rtc.Now.Hour()), nil
		}

		// Top bit must be set in 12-hour format.
		// This is such a frustrating way to represent time.
		hour := reg.Rtc.Val(reg.Rtc.Now.Hour() % 12)
		if reg.Rtc.Now.Hour() >= 12 {
			return 0x80 | hour, nil
		}
		return hour, nil

	case RtcWeekday:
		return reg.Rtc.Val(int(reg.Now.Weekday())), nil

	case RtcDay:
		return reg.Rtc.Val(reg.Now.Day()), nil

	case RtcMonth:
		return reg.Rtc.Val(int(reg.Now.Month())), nil

	case RtcYear:
		return reg.Rtc.Val(reg.Now.Year()), nil

	case RtcStatusA:
		return uint64(reg.Rtc.StatusA), nil

	case RtcStatusB:
		return uint64(reg.Rtc.StatusB), nil

	case RtcIntr:
		return 0, nil

	case RtcStatusD:
		return RtcStatusDPWR, nil
	}

	return uint64(math.MaxUint64), nil
}

func (reg *RtcData) Write(offset uint64, size uint, value uint64) error {

	val := uint8(value)

	switch reg.Rtc.Addr {

	case RtcStatusA:
		reg.Rtc.StatusA = val & ^uint8(RtcStatusATUP)
		break

	case RtcStatusB:
		reg.Rtc.StatusB = val
		break

	case RtcIntr:
		// Ignore.
		break

	case RtcStatusD:
		// Ignore.
		break

	case RtcSecondAlarm:
		reg.Rtc.AlarmSecond = val
		break

	case RtcMinuteAlarm:
		reg.Rtc.AlarmMinute = val
		break

	case RtcHourAlarm:
		reg.Rtc.AlarmHour = val
		break
	}

	return nil
}

func NewRtc(info *DeviceInfo) (Device, error) {

	// Create the rtc.
	rtc := new(Rtc)
	rtc.PioDevice.Offset = 0x70
	rtc.PioDevice.IoMap = IoMap{
		// Our configuration ports.
		MemoryRegion{0, 1}: &RtcAddr{Rtc: rtc},
		MemoryRegion{1, 1}: &RtcData{Rtc: rtc},
	}

	return rtc, rtc.init(info)
}

func (rtc *Rtc) Attach(vm *platform.Vm, model *Model) error {

	// Update the time.
	var defaultTime time.Time
	if rtc.Now != defaultTime {
		rtc.Tick(false)
	} else {
		rtc.Tick(true)
	}

	return rtc.PioDevice.Attach(vm, model)
}
