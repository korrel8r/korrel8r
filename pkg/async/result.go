// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package async provides goroutine-safe collection types.
package async

import (
	"sync"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
)

var _ result.Result = (*Result)(nil)

// Result is a goroutine-safe [result.Result] with blocking Wait support.
type Result struct {
	mu     sync.Mutex
	cond   *sync.Cond
	inner  result.Result
	closed bool
}

func New(class korrel8r.Class) *Result {
	r := &Result{inner: result.New(class)}
	r.cond = sync.NewCond(&r.mu)
	return r
}

func (r *Result) Add(o korrel8r.Object) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	ok := r.inner.Add(o)
	if ok {
		r.cond.Broadcast()
	}
	return ok
}

func (r *Result) Append(objects ...korrel8r.Object) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inner.Append(objects...)
	if len(objects) > 0 {
		r.cond.Broadcast()
	}
}

func (r *Result) List() []korrel8r.Object {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inner.List()
}

// Wait blocks until len(List()) > n or the Result is closed.
// Returns List()[n:] or nil if closed with no new elements.
func (r *Result) Wait(n int) []korrel8r.Object {
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
func (r *Result) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	r.cond.Broadcast()
}
