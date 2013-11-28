package machine

import (
    "novmm/platform"
)

type PioEvent struct {
    *platform.ExitPio
}

func (pio PioEvent) Size() uint {
    return pio.ExitPio.Size()
}

func (pio PioEvent) GetData() uint64 {
    return *pio.ExitPio.Data()
}

func (pio PioEvent) SetData(val uint64) {
    *pio.ExitPio.Data() = val
}

func (pio PioEvent) IsWrite() bool {
    return pio.ExitPio.IsOut()
}
