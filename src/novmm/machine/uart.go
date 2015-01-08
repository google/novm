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
	"os"
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
	*Uart
}

type UartIntr struct {
	*Uart
}

func (uart *UartData) Read(offset uint64, size uint) (uint64, error) {
	if uart.Lcr.Value&UartLcrDLAB != 0 {
		return uart.Dll.Read(offset, size)
	}

	// No data available.
	return math.MaxUint64, nil
}

func (uart *UartData) Write(offset uint64, size uint, value uint64) error {
	if uart.Lcr.Value&UartLcrDLAB != 0 {
		return uart.Dll.Write(offset, size, value)
	}

	// Ignore return value.
	os.Stdout.Write([]byte{byte(value)})
	return nil
}

func (uart *UartIntr) Read(offset uint64, size uint) (uint64, error) {
	if uart.Lcr.Value&UartLcrDLAB != 0 {
		return uart.Dlh.Read(offset, size)
	}

	return uart.Ier.Read(offset, size)
}

func (uart *UartIntr) Write(offset uint64, size uint, value uint64) error {
	if uart.Lcr.Value&UartLcrDLAB != 0 {
		return uart.Dlh.Write(offset, size, value)
	}

	return uart.Ier.Write(offset, size, value)
}

type Uart struct {
	PioDevice

	// Registers.
	Ier Register `json:"ier"`
	Iir Register `json:"iir"`
	Lcr Register `json:"lcr"`
	Mcr Register `json:"mcr"`
	Lsr Register `json:"lsr"`
	Msr Register `json:"msr"`
	Fcr Register `json:"fcr"`
	Scr Register `json:"scr"`
	Dll Register `json:"dll"`
	Dlh Register `json:"dlh"`

	// Our allocated interrupt.
	InterruptNumber platform.Irq `json:"interrupt"`
}

func NewUart(info *DeviceInfo) (Device, error) {

	// Create the uart.
	uart := new(Uart)

	// Create our IOmap.
	uart.PioDevice.IoMap = IoMap{
		// Our configuration ports.
		MemoryRegion{0, 1}: &UartData{Uart: uart},
		MemoryRegion{1, 1}: &UartIntr{Uart: uart},
		MemoryRegion{2, 1}: &uart.Iir, // Interrupt identification.
		MemoryRegion{3, 1}: &uart.Lcr, // Line control register.
		MemoryRegion{4, 1}: &uart.Mcr, // Modem control register.
		MemoryRegion{5, 1}: &uart.Lsr, // Line status register.
		MemoryRegion{6, 1}: &uart.Msr, // Modem status register.
		MemoryRegion{7, 1}: &uart.Scr, // Scratch register.
	}

	// Set our readonly bits.
	uart.Lsr.readonly = 0xff
	uart.Msr.readonly = 0xff
	uart.Msr.readclr = 0x0f
	uart.Mcr.readonly = 0x1f

	// We're always ready for data.
	uart.Lsr.Value = UartLsrTEMT | UartLsrRXRDY | UartLsrTHRE
	uart.Lsr.readonly = uart.Lsr.Value

	// Clear the OE bit on read.
	uart.Lsr.readclr = UartLsrOE

	// Set our divisor.
	divisor := uint64(UartDefaultRclk / UartDefaultBaud / 16)
	uart.Dll.Value = divisor
	uart.Dlh.Value = divisor >> 16

	return uart, uart.init(info)
}

func (uart *Uart) getInterruptStatus() uint8 {
	if uart.Lsr.Value&UartLsrOE != 0 && uart.Ier.Value&UartIerERLS != 0 {
		return UartIirRLS
	} else if uart.Msr.Value&UartMsrMASK != 0 && uart.Ier.Value&UartIerEMSC != 0 {
		return UartIirMLSC
	}

	return UartIirNOPEND
}
