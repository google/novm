package machine

import (
    "novmm/platform"
)

func (model *Model) Handle(
    vm *platform.Vm,
    cache *IoCache,
    handler *IoHandler,
    ioevent IoEvent,
    addr platform.Paddr) error {

    if handler != nil {

        // Our offset from handler start.
        offset := addr.OffsetFrom(handler.start)

        // Submit our function.
        err := handler.queue.Submit(ioevent, offset)

        // Should we save this request?
        if ioevent.IsWrite() && err == SaveIO {
            err = cache.save(
                vm,
                addr,
                ioevent.Size(),
                ioevent.GetData(),
                func() error {
                    return handler.queue.Submit(ioevent, offset)
                })
        }

        // Return to our vcpu.
        return err

    } else if !ioevent.IsWrite() {

        // Invalid reads return all 1's.
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

func (model *Model) HandlePio(
    vm *platform.Vm,
    event *platform.ExitPio) error {

    handler := model.pio_cache.lookup(event.Port())
    ioevent := &PioEvent{event}
    return model.Handle(vm, model.pio_cache, handler, ioevent, event.Port())
}

func (model *Model) HandleMmio(
    vm *platform.Vm,
    event *platform.ExitMmio) error {

    handler := model.mmio_cache.lookup(event.Addr())
    ioevent := &MmioEvent{event}
    return model.Handle(vm, model.mmio_cache, handler, ioevent, event.Addr())
}
