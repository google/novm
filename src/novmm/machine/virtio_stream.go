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

//
// VirtioStream --
//
// The VirtioStream is a simple wrapper
// around the buffer. It essentially provides
// a remember offset to make it easy for encode
// & decode operations. Nothing fancy.
//
// This interface will silently drop most
// errors in order to make using it as convenient
// as possible. It is the duty of the caller to
// checks ReadLeft() / WriteLeft() and ensure that
// it is always >= 0, otherwise they will know
// that they overran the buffer.
//

type VirtioStream struct {
	*VirtioBuffer

	// Our read offset in the buffer.
	read_offset int

	// Our write offset in the buffer.
	write_offset int

	// A bookmark for rewind.
	bookmark int
}

func (stream *VirtioStream) ReadLeft() int {
	return stream.VirtioBuffer.Length() - stream.read_offset
}

func (stream *VirtioStream) ReadRewind() {
	stream.read_offset = stream.bookmark
}

func (stream *VirtioStream) WriteLeft() int {
	return stream.VirtioBuffer.Length() - stream.write_offset
}

func (stream *VirtioStream) WriteRewind() {
	stream.write_offset = stream.bookmark
}

func (stream *VirtioStream) ReadN(size int) uint64 {

	// Are we finished?
	if stream.ReadLeft() < size {
		stream.read_offset += size
		return 0
	}

	// Map our chunk.
	ram := &Ram{stream.Map(stream.read_offset, size)}

	// Check for boundary.
	if ram.Size() < size {
		n1 := stream.ReadN(size / 2)
		n2 := stream.ReadN(size / 2)
		return (n2 << (uint(size) << 2)) | n1
	}

	// All there.
	stream.read_offset += size

	switch size {
	case 8:
		return ram.Get64(0)
	case 4:
		return uint64(ram.Get32(0))
	case 2:
		return uint64(ram.Get16(0))
	case 1:
		return uint64(ram.Get8(0))
	}

	return 0
}

func (stream *VirtioStream) WriteN(size int, value uint64) {

	// Are we finished?
	if stream.WriteLeft() < size {
		stream.write_offset += size
		return
	}

	// Map our chunk.
	ram := &Ram{stream.Map(stream.write_offset, size)}

	// Check for boundary.
	if ram.Size() < size {
		stream.WriteN(size/2, value)
		stream.WriteN(size/2, value>>(uint(size)<<2))
	}

	stream.write_offset += size

	// All there.
	switch size {
	case 8:
		ram.Set64(0, value)
	case 4:
		ram.Set32(0, uint32(value))
	case 2:
		ram.Set16(0, uint16(value))
	case 1:
		ram.Set8(0, uint8(value))
	}
}

func (stream *VirtioStream) Read8() uint8 {
	return uint8(stream.ReadN(1))
}

func (stream *VirtioStream) Read16() uint16 {
	return uint16(stream.ReadN(2))
}

func (stream *VirtioStream) Read32() uint32 {
	return uint32(stream.ReadN(4))
}

func (stream *VirtioStream) Read64() uint64 {
	return uint64(stream.ReadN(8))
}

func (stream *VirtioStream) ReadBytes(length int) []byte {

	ram := stream.Map(stream.read_offset, length)
	if ram == nil {
		// Give back an empty buffer.
		stream.read_offset += length
		return make([]byte, length, length)
	}

	// Copy the prefix.
	value := make([]byte, len(ram), len(ram))
	copy(value, ram)
	stream.read_offset += len(ram)

	if len(ram) < length {
		suffix := stream.ReadBytes(length - len(ram))
		return append(value, suffix...)
	}

	return value
}

func (stream *VirtioStream) ReadString() string {
	length := stream.Read16()
	return string(stream.ReadBytes(int(length)))
}

func (stream *VirtioStream) Write8(value uint8) {
	stream.WriteN(1, uint64(value))
}

func (stream *VirtioStream) Write16(value uint16) {
	stream.WriteN(2, uint64(value))
}

func (stream *VirtioStream) Write32(value uint32) {
	stream.WriteN(4, uint64(value))
}

func (stream *VirtioStream) Write64(value uint64) {
	stream.WriteN(8, value)
}

func (stream *VirtioStream) WriteBytes(data []byte) {

	// Map our segment.
	ram := stream.Map(stream.write_offset, len(data))
	if ram == nil {
		stream.write_offset += len(data)
		return
	}

	// Copy the prefix.
	if len(ram) > len(data) {
		copy(ram, data)
		stream.write_offset += len(data)
	} else {
		copy(ram, data[:len(ram)])
		stream.write_offset += len(ram)
	}

	// Copy the suffix.
	if len(ram) < len(data) {
		stream.WriteBytes(data[len(ram):])
	}
}

func (stream *VirtioStream) WriteString(value string) {
	data := []byte(value)
	stream.Write16(uint16(len(data)))
	stream.WriteBytes(data)
}

func (stream *VirtioStream) ReadFromFd(
	fd int,
	offset int64,
	length int) (int, error) {

	length, err := stream.VirtioBuffer.PRead(
		fd,
		offset,
		stream.write_offset,
		length)

	if err == nil {
		stream.write_offset += length
	}

	return length, err
}

func (stream *VirtioStream) WriteToFd(
	fd int,
	offset int64,
	length int) (int, error) {

	length, err := stream.VirtioBuffer.PWrite(
		fd,
		offset,
		stream.read_offset,
		length)
	if err == nil {
		stream.read_offset += length
	}

	return length, err
}

func NewVirtioStream(
	buffer *VirtioBuffer,
	bookmark int) *VirtioStream {

	vs := new(VirtioStream)
	vs.VirtioBuffer = buffer
	vs.bookmark = bookmark
	vs.read_offset = bookmark
	vs.write_offset = bookmark
	return vs
}
