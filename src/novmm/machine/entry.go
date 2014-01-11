package machine

import (
    "novmm/platform"
)

func (model *Model) Handle(
    handler *IoHandler,
    ioevent IoEvent,
    addr platform.Paddr) error {

    if handler != nil {
        // Send it to the queue.
        return handler.queue.Submit(
            ioevent,
            addr.OffsetFrom(handler.start))

    } else if !ioevent.IsWrite() {
        switch ioevent.Size() {
        case 1:
            ioevent.SetData(0xff)
        case 2:
            ioevent.SetData(0xffff)
        case 4:
            ioevent.SetData(0xffffffff)
        case 8:
            ioevent.SetData(0xffffffffffffffff)
        }
    }

    return nil
}

func (model *Model) HandlePio(event *platform.ExitPio) error {

    handler := model.pio_cache.lookup(event.Port())
    ioevent := &PioEvent{event}
    return model.Handle(handler, ioevent, event.Port())
}

func (model *Model) HandleMmio(event *platform.ExitMmio) error {

    handler := model.mmio_cache.lookup(event.Addr())
    ioevent := &MmioEvent{event}
    return model.Handle(handler, ioevent, event.Addr())
}
