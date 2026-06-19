// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package async provides goroutine-safe collection types.
package result

import (
	"sync"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var _ Result = (*SyncResult)(nil)

// SyncResult is a concurrent-safe [Result] with blocking [Wait].
type SyncResult struct {
	mu     sync.Mutex
	cond   *sync.Cond
	inner  Result
	closed bool
}

func NewSyncResult(class korrel8r.Class) *SyncResult {
	r := &SyncResult{inner: New(class)}
	r.cond = sync.NewCond(&r.mu)
	return r
}

func (r *SyncResult) Add(o korrel8r.Object) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	ok := r.inner.Add(o)
	if ok {
		r.cond.Broadcast()
	}
	return ok
}

func (r *SyncResult) Append(objects ...korrel8r.Object) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inner.Append(objects...)
	if len(objects) > 0 {
		r.cond.Broadcast()
	}
}

func (r *SyncResult) List() []korrel8r.Object {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inner.List()
}

// Wait blocks until len(List()) > n or the SyncResult is closed.
// Returns List()[n:] or nil if closed with no new elements.
func (r *SyncResult) Wait(n int) []korrel8r.Object {
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(r.inner.List()) <= n && !r.closed {
		r.cond.Wait()
	}
	list := r.inner.List()
	if len(list) <= n {
		return nil
	}
	return list[n:]
}

// Close signals that no more objects will be added, unblocking all waiters.
func (r *SyncResult) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	r.cond.Broadcast()
}
