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
	"novmm/platform"
)

//
// I/O cache --
//
// Our I/O cache stores paddr => handler mappings.
//
// When a device returns a SaveIO error, we actually try to
// setup an EventFD for that specific addr and value. This
// will avoid having to go through the cache every time. We
// do this only after accruing sufficient hits however, in
// order to avoid wasting eventfds on devices that only hit
// a few times (like an unused NIC, for example).
//
// See eventfd.go for the save() function where this is done.

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

func NewIoCache(handlers []IoHandlers, is_pio bool) *IoCache {
	return &IoCache{
		handlers: handlers,
		memory:   make(map[platform.Paddr]*IoHandler),
		hits:     make(map[platform.Paddr]uint64),
		is_pio:   is_pio,
	}
}
