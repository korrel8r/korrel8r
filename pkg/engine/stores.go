// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"sync"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var (
	_ korrel8r.Store = &store{}
	_ korrel8r.Store = &stores{}
)

// store is a wrapper to (re-)create a store on demand from its configuration.
type store struct {
	lock sync.Mutex

	Original config.Store   // Original template configuration to create the store.
	Expanded config.Store   // Expanded template used for last creation attempt.
	Store    korrel8r.Store // Store client. Nil if store needs to be re-created.
	Err      error          // Last non-nil error from Store.Get() or Domain.Store()
	ErrCount int            // Count of errors from Store.Get() and Domain.Store()

	domain korrel8r.Domain
	expand func(string) (string, error) // Expand template configuration
}

func (s *store) Domain() korrel8r.Domain { return s.domain }

// Get (re-)creates the store as required. Concurrent safe.
func (s *store) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) (err error) {
	s.lock.Lock() // Lock for duration of Get() - serialize Get per store.
	defer s.lock.Unlock()
	if err := s.ensure(); err != nil {
		return err
	}
	err = s.Store.Get(ctx, q, constraint, result)
	if err != nil {
		s.Err = err
		s.ErrCount++
		if s.Original != nil { // Only re-create if there is some configuration.
			// Close the broken store if it is an io.Closer()
			if c, ok := s.Store.(io.Closer); ok {
				_ = c.Close()
			}
			s.Store = nil // Re-create on next use
		}
	}
	return err
}

// Ensure the store is connected.
func (s *store) Ensure() (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.ensure()
}

// ensure is unsafe, must be called with lock held, via Ensure()
func (s *store) ensure() (err error) {
	if s.Store != nil {
		return nil // Already exists.
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
		if v, err = s.expand(v); err != nil {
			return err
		}
		s.Expanded[k] = v
	}
	// Create the store
	if _, ok := s.Expanded[config.StoreKeyMock]; ok {
		// Special case for mock store, any domain can have a mock store.
		s.Store, err = mock.NewStoreConfig(s.domain, s.Expanded)
	} else { // Domain-specific store
		s.Store, err = s.domain.Store(s.Expanded)
	}
	if err != nil {
		s.Store = nil
	}
	return err
}

// stores contains multiple configured stores and iterates over them in Get.
type stores struct {
	domain korrel8r.Domain
	stores []*store
	expand func(string) (string, error)
}

func newStores(e *Engine, d korrel8r.Domain) *stores {
	return &stores{
		domain: d,
		stores: []*store{},
		expand: func(s string) (string, error) { return e.execTemplate(s, nil) },
	}
}

func (ss *stores) Domain() korrel8r.Domain { return ss.domain }

func (ss *stores) Add(newStore *store) error {
	newStore.expand = ss.expand
	// Check for duplicate configuration
	if newStore.Original != nil && slices.ContainsFunc(ss.stores,
		func(s *store) bool { return reflect.DeepEqual(s.Original, newStore.Original) }) {
		return fmt.Errorf("duplicate store configuration: %v", newStore.Original)
	}
	ss.stores = append(ss.stores, newStore)
	return nil
}

func (ss *stores) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	var (
		errs error
		ok   bool
	)
	for _, s := range ss.stores {
		// Iterate over stores and accumulate all results.
		err := s.Get(ctx, q, constraint, result)
		ok = (err == nil) || ok       // Remember if any call succeeds.
		errs = errors.Join(errs, err) // Collect errors in case of failure.
	}
	if ok { // If any call succeeded, this is a success
		return nil
	}
	return errs
}

// Configs returns the expanded configurations for each store.
func (ss *stores) Configs() (ret []config.Store) {
	for _, s := range ss.stores {
		sc := maps.Clone(s.Expanded)
		if s.Err != nil {
			sc[config.StoreKeyError] = s.Err.Error()
		}
		if s.ErrCount > 0 {
			sc[config.StoreKeyErrorCount] = strconv.Itoa(s.ErrCount)
		}
		ret = append(ret, sc)
	}
	return ret
}

// Ensure calls [configuredStore.Ensure] on all configured stores.
func (ss *stores) Ensure() {
	for _, s := range ss.stores {
		// Not an error if create fails, will be registered in stores.
		_ = s.Ensure()
	}
}
