// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"errors"
	"sync"

	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Errors is a goroutine-safe collection of errors.
type Errors struct {
	m    sync.Mutex
	err  error
	ok   int
	seen unique.Set[string]
}

func NewErrors() *Errors {
	return &Errors{seen: unique.Set[string]{}}
}

func (e *Errors) Add(err error) bool {
	e.m.Lock()
	defer e.m.Unlock()
	if err == nil {
		e.ok++
	} else if !e.seen.Has(err.Error()) {
		e.err = errors.Join(e.err, err)
		e.seen.Add(err.Error())
		return true
	}
	return false
}

// Err returns the resulting error which may be a PartialError.
// Must not be called concurrently.
func (e *Errors) Err() error {
	switch {
	case e.err == nil:
		return nil
	case e.ok > 0:
		return &PartialError{e.err}
	default:
		return e.err
	}
}

// PartialError indicates some errors were encountered but there are still some results.
type PartialError struct{ Err error }

func (e *PartialError) Error() string {
	return errors.Join(errors.New("Results may be incomplete, there were errors"), e.Err).Error()
}

// IsPartial returns true if err contains a *[PartialError] indicating partial content available.
func IsPartial(err error) bool {
	var pe *PartialError
	return errors.As(err, &pe)
}

var _ error = &PartialError{}
