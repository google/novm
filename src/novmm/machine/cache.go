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
//
// That was sort of the long-term idea for the cache.

type IoCache struct {
    handlers []*IoHandlers
    memory   map[platform.Paddr]*IoHandler
    hits     map[platform.Paddr]uint64
}

func (cache *IoCache) lookup(addr platform.Paddr) *IoHandler {

    handler, ok := cache.memory[addr]
    if ok {
        cache.hits[addr] += 1
        return handler
    }

    // See if we can find a matching device.
    for _, handlers := range cache.handlers {
        for port_region, handler := range *handlers {
            if port_region.Contains(addr, 1) {
                cache.memory[addr] = handler
                cache.hits[addr] += 1
                return handler
            }
        }
    }

    // Nothing found.
    log.Fatal("Cache miss!? Shouldn't happen.")
    cache.memory[addr] = nil
    return nil
}

func NewIoCache(handlers []*IoHandlers) *IoCache {
    return &IoCache{
        handlers: handlers,
        memory:   make(map[platform.Paddr]*IoHandler),
        hits:     make(map[platform.Paddr]uint64),
    }
}
