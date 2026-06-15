// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// TTL is a goroutine-safe cache with per-entry time-to-live expiry.
// Expired entries are cleaned up passively on each access.
type TTL[K comparable, V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[K]entry[V]
}

// NewTTL creates a new TTL cache. Entries expire after the given duration.
func NewTTL[K comparable, V any](ttl time.Duration) *TTL[K, V] {
	return &TTL[K, V]{
		ttl:     ttl,
		entries: make(map[K]entry[V]),
	}
}

// Get returns the value for key and true if found and not expired, or the zero value and false otherwise.
func (c *TTL[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictExpired()
	if e, ok := c.entries[key]; ok {
		return e.value, true
	}
	var zero V
	return zero, false
}

// Put adds or replaces an entry in the cache.
func (c *TTL[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictExpired()
	c.entries[key] = entry[V]{value: value, expiresAt: time.Now().Add(c.ttl)}
}

// Clear removes all entries from the cache.
func (c *TTL[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.entries)
}

func (c *TTL[K, V]) evictExpired() {
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
}
