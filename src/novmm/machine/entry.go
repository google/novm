package machine

import (
    "novmm/platform"
)

func (model *Model) HandlePio(event *platform.ExitPio) error {

    // NOTE: We always expect to have a handler,
    // and there's a InvalidDevice that will pick
    // up any invalid addresses at the end of pack.
    handler := model.pio_cache.lookup(event.Port())
    io_event := &PioEvent{event}

    // Send it to the queue.
    return handler.queue.Submit(
        io_event,
        event.Port().OffsetFrom(handler.MemoryRegion.Start))
}

func (model *Model) HandleMmio(event *platform.ExitMmio) error {

    // See NOTE above.
    handler := model.mmio_cache.lookup(event.Addr())
    io_event := &MmioEvent{event}

    // Send it to the queue.
    return handler.queue.Submit(
        io_event,
        event.Addr().OffsetFrom(handler.MemoryRegion.Start))
}
