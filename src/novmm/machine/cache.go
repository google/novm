package machine

import (
    "log"
    "novmm/platform"
)

//
// I/O cache --
//
// Our I/O cache stores paddr => handler mappings.
//
// Eventually it would be nice to detect the devices that
// get hit the most, and create an EventFd that would wait
// on that device in order to prevent bouncing in and out
// of the kernel. But maybe we can do that for all devices.

type IoCache struct {

    // Our set of I/O handlers.
    handlers []IoHandlers

    // Our I/O cache.
    memory map[platform.Paddr]*IoHandler

    // Our hits.
    hits map[platform.Paddr]uint64

    // Is this a Pio cache?
    is_pio bool
}

func (cache *IoCache) lookup(addr platform.Paddr) *IoHandler {

    handler, ok := cache.memory[addr]
    if ok {
        cache.hits[addr] += 1
        return handler
    }

    // See if we can find a matching device.
    for _, handlers := range cache.handlers {
        for port_region, handler := range handlers {
            if port_region.Contains(addr, 1) {
                cache.memory[addr] = handler
                cache.hits[addr] += 1
                return handler
            }
        }
    }

    // Nothing found.
    return nil
}

func (cache *IoCache) save(
    vm *platform.Vm,
    addr platform.Paddr,
    size uint,
    value uint64,
    fn func() error) error {

    // Do we have sufficient hits?
    if cache.hits[addr] < 100 {
        return nil
    }

    // Bind an eventfd.
    // NOTE: NewBoundEventFd() may return nil, nil
    // in which case we don't have eventfds enabled.
    boundfd, err := vm.NewBoundEventFd(addr, size, cache.is_pio, true, value)
    if err != nil || boundfd == nil {
        return err
    }

    log.Printf(
        "eventfd [addr=%08x size=%x is_pio=%t value=%08x]",
        addr,
        size,
        cache.is_pio,
        value)

    // Run our function.
    go func() {
        for {
            // Wait for the next event.
            _, err := boundfd.Wait()
            if err != nil {
                break
            }

            // Call our function.
            // We disregard any errors on this
            // function, since there's nothing
            // that we can actually do here.
            fn()
        }

        // Finished with the eventfd.
        boundfd.Close()
    }()

    // Success.
    return nil
}

func NewIoCache(handlers []IoHandlers, is_pio bool) *IoCache {
    return &IoCache{
        handlers: handlers,
        memory:   make(map[platform.Paddr]*IoHandler),
        hits:     make(map[platform.Paddr]uint64),
        is_pio:   is_pio,
    }
}
