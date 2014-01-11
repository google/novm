package machine

type NvRam []byte

func (nvram *NvRam) Set8(offset int, data uint8) {
    (*nvram)[offset] = byte(data)
}

func (nvram *NvRam) Get8(offset int) uint8 {
    return (*nvram)[offset]
}

func (nvram *NvRam) Set16(offset int, data uint16) {
    (*nvram)[offset] = byte(data & 0xff)
    (*nvram)[offset+1] = byte((data >> 8) & 0xff)
}

func (nvram *NvRam) Get16(offset int) uint16 {
    return (uint16((*nvram)[offset]) |
        (uint16((*nvram)[offset+1]) << 8))
}

func (nvram *NvRam) Set32(offset int, data uint32) {
    (*nvram)[offset] = byte(data & 0xff)
    (*nvram)[offset+1] = byte((data >> 8) & 0xff)
    (*nvram)[offset+2] = byte((data >> 16) & 0xff)
    (*nvram)[offset+3] = byte((data >> 24) & 0xff)
}

func (nvram *NvRam) Get32(offset int) uint32 {
    return (uint32((*nvram)[offset]) |
        (uint32((*nvram)[offset+1]) << 8) |
        (uint32((*nvram)[offset+2]) << 16) |
        (uint32((*nvram)[offset+3]) << 24))
}
