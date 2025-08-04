// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

import (
	"errors"
	"sync"
)

// Errors collects errors with unique messages, discards duplicates.
// A Zero Errors can be used immediately.
// Concurrent safe.
type Errors struct {
	err  error
	seen Set[string]
	m    sync.Mutex
}

// Err returns a composite error created using [errors.Join] or nil.
// Concurrent safe
func (e *Errors) Err() error {
	e.m.Lock()
	defer e.m.Unlock()
	return e.err
}

// Add an error if it has not already been added.
// Returns true if the error is unique.
// Concurrent safe
func (e *Errors) Add(err error) bool {
	e.m.Lock()
	defer e.m.Unlock()
	if err != nil {
		if e.seen == nil {
			e.seen = Set[string]{}
		}
		s := err.Error()
		if !e.seen.Has(s) {
			e.seen.Add(s)
			e.err = errors.Join(e.err, err)
			return true
		}
	}
	return false
}
