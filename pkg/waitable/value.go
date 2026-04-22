// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package waitable

import (
	"sync"
)

// Value holds a value of type T that can be [Set] at any time.
// [Get] can be used to grab the "current" value and be notified of updates,
// but is not guaranteed to see every value that is [Set]
type Value[T any] struct {
	mu     sync.Mutex
	val    T
	update chan struct{}
}

func NewValue[T any](initial T) *Value[T] {
	return &Value[T]{val: initial, update: make(chan struct{})}
}

// Get returns the current value.
func (c *Value[T]) Get() T {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.val
}

// GetChan returns the current value, and a channel that will close when Value.Set is called.
func (c *Value[T]) GetChan() (T, chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.val, c.update
}

// Set sets the current value and closes the waiting channel returned by [Get].
func (c *Value[T]) Set(v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val = v
	close(c.update)
	c.update = make(chan struct{})
}
