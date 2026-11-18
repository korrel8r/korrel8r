// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE
package traverse

import (
	"fmt"
	"runtime"
	"slices"
	"sync"
	"weak"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// A neighborhood's routing depends only on the engine, start class and depth.
// Entries are read-only and bounded; weak keys avoid directly retaining
// engines, and GC cleanups remove their routing after collection.
type neighborRoutingCache struct {
	mu       sync.Mutex
	entries  []neighborRoutingEntry // FIFO, oldest first.
	inFlight map[neighborRoutingKey]*routingBuild
}

type routingBuild struct {
	ready   chan struct{}
	routing []nodeRouting
	err     error
}

type neighborRoutingKey struct {
	engine weak.Pointer[engine.Engine]
	start  int64
	depth  int
}

type neighborRoutingEntry struct {
	key     neighborRoutingKey
	routing []nodeRouting
}

type engineCleanup struct {
	cache *neighborRoutingCache
	key   weak.Pointer[engine.Engine]
}

func (arg engineCleanup) remove() { arg.cache.removeEngine(arg.key) }

const maxNeighborRoutings = 16

var neighborRoutings neighborRoutingCache

func (c *neighborRoutingCache) removeEngine(key weak.Pointer[engine.Engine]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = slices.DeleteFunc(c.entries, func(entry neighborRoutingEntry) bool {
		return entry.key.engine == key
	})
}

func (c *neighborRoutingCache) get(e *engine.Engine, start korrel8r.Class, depth int) ([]nodeRouting, error) {
	data := e.GraphData()
	startID, ok := data.NodeID(start)
	if !ok {
		return nil, fmt.Errorf("class not found in graph: %v", start)
	}
	key := neighborRoutingKey{weak.Make(e), startID, depth}
	defer runtime.KeepAlive(e)
	return c.getOrBuild(e, key, func() ([]nodeRouting, error) {
		scope, err := neighborScope(data, start, depth)
		if err != nil {
			return nil, err
		}
		routing := prepareRouting(e, data, scope)
		// run initializes the start class even for an empty scope (e.g. depth 0).
		// Initialize it before publication so run never writes to shared routing.
		initRoutingNode(e, &routing[startID], data.Class(startID))
		return routing, nil
	})
}

// getOrBuild coalesces cold misses for one key while builds for different keys
// and warm cache hits proceed independently. The channel publishes the result
// (including an error) to callers waiting for the same build.
func (c *neighborRoutingCache) getOrBuild(e *engine.Engine, key neighborRoutingKey, build func() ([]nodeRouting, error)) ([]nodeRouting, error) {
	c.mu.Lock()
	for _, entry := range c.entries {
		if entry.key == key {
			c.mu.Unlock()
			return entry.routing, nil
		}
	}
	if pending := c.inFlight[key]; pending != nil {
		c.mu.Unlock()
		<-pending.ready
		return pending.routing, pending.err
	}
	if c.inFlight == nil {
		c.inFlight = make(map[neighborRoutingKey]*routingBuild)
	}
	pending := &routingBuild{ready: make(chan struct{})}
	c.inFlight[key] = pending
	c.mu.Unlock()

	routing, err := build()
	c.mu.Lock()
	if err == nil {
		if !slices.ContainsFunc(c.entries, func(entry neighborRoutingEntry) bool {
			return entry.key.engine == key.engine
		}) {
			// The cleanup argument must not contain a strong reference to e.
			// Re-entry after eviction can register an additional harmless cleanup.
			runtime.AddCleanup(e, engineCleanup.remove, engineCleanup{c, key.engine})
		}
		if len(c.entries) == maxNeighborRoutings {
			c.entries = slices.Delete(c.entries, 0, 1)
		}
		c.entries = append(c.entries, neighborRoutingEntry{key, routing})
	}
	pending.routing, pending.err = routing, err
	delete(c.inFlight, key)
	close(pending.ready)
	c.mu.Unlock()
	return routing, err
}
