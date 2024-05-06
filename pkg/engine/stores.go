// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

type store struct {
	lock sync.Mutex

	Original config.Store   // Original templated config to create the store.
	Expanded config.Store   // Expanded template used for last creation attempt.
	Store    korrel8r.Store // Store client. Nil if store needs to be re-created.
	Err      error          // Last non-nil error from Store.Get() or Domain.Store()
	ErrCount int            // Count of errors from Store.Get() and Domain.Store()
}

func (s *store) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender, expand func(string) (string, error)) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.ensure(q.Class().Domain(), expand); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			log.V(2).Info("Get error", "error", err, "query", q)
			s.Err = err
			s.ErrCount++
		}
	}()
	if err := s.Store.Get(ctx, q, constraint, result); err != nil {
		if s.Original != nil { // Only re-create if there is some configuration.
			// Close the broken store if it is an io.Closer()
			if c, ok := s.Store.(io.Closer); ok {
				_ = c.Close()
			}
			s.Store = nil // Re-create on next use
		}
		return err
	}
	return nil
}

// Ensure the store is connected.
func (s *store) Ensure(d korrel8r.Domain, expand func(string) (string, error)) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.ensure(d, expand)
}

// ensure is unsafe, must be called with lock held.
func (s *store) ensure(d korrel8r.Domain, expand func(string) (string, error)) (err error) {
	if s.Store != nil {
		return nil // No need to create or re-create.
	}
	defer func() {
		if err != nil {
			s.Err = err
			s.ErrCount++
			log.V(2).Info("Store error", "error", err, "config", s.Original)
		}
	}()
	// Expand the store config each time - the results may change.
	s.Expanded = config.Store{}
	for k, v := range s.Original {
		vv, err := expand(v)
		if err != nil {
			return err
		}
		s.Expanded[k] = vv // FIXME CONCURRENT HERE
	}
	// Create the store
	ns, err := d.Store(s.Expanded)
	if err == nil {
		s.Store = ns
	}
	return err
}

type stores []*store

func (ss stores) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender, expand func(string) (string, error)) error {
	var (
		errs error
		ok   bool
	)
	for _, s := range ss {
		err := s.Get(ctx, q, constraint, result, expand)
		ok = (err == nil) || ok       // If any call succeeds consider this a success.
		errs = errors.Join(errs, err) // Collect errors in case of failure
	}
	if ok {
		errs = nil
	}
	return errs
}
