package machine

import (
    "novmm/platform"
)

type MmioEvent struct {
    *platform.ExitMmio
}

func (mmio MmioEvent) Size() uint {
    return mmio.ExitMmio.Length()
}

func (mmio MmioEvent) GetData() uint64 {
    return *mmio.ExitMmio.Data()
}

func (mmio MmioEvent) SetData(val uint64) {
    *mmio.ExitMmio.Data() = val
}

func (mmio MmioEvent) IsWrite() bool {
    return mmio.ExitMmio.IsWrite()
}
