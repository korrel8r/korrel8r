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
	"text/template"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

var (
	_ korrel8r.Store = &storeHolder{}
	_ korrel8r.Store = &storeHolder{}
)

// storeHolder is a wrapper to (re-)create a store on demand from its configuration.
// Keeps track of errors connecting to the store for debugging.
// Concurrent safe.
type storeHolder struct {
	lock sync.Mutex

	Original   config.Store   // Original template configuration to create the store.
	Expanded   config.Store   // Expanded template used for last creation attempt.
	Store      korrel8r.Store // Store client. Nil if store needs to be created.
	LastErr    error          // Last error connecting to the store.
	ErrCount   int            // Count of errors connecting to the store.
	retryAfter time.Time      // Don't attempt re-creation before this time.
	Engine     *Engine

	domain korrel8r.Domain // Must be a method to fit Store interface.
}

// wrap wraps a [config.Store] or a [korrel8r.Store] as a *[storeHolder]
// Exactly one of sc and s must be non-nil.
func wrap(e *Engine, sc config.Store, s korrel8r.Store) (*storeHolder, error) {
	var d korrel8r.Domain
	if s != nil {
		d = s.Domain()
	} else {
		var err error
		d, err = e.Domain(sc[config.StoreKeyDomain])
		if err != nil {
			return nil, err
		}
	}
	return &storeHolder{Engine: e, Original: sc, Expanded: nil, Store: s, domain: d}, nil
}

func (s *storeHolder) Domain() korrel8r.Domain { return s.domain }

// recordErrorLH records a store error and resets the store for re-creation.
// Must be called with the lock held.
func (s *storeHolder) recordErrorLH(err error) {
	if err != nil {
		s.LastErr = err
		s.ErrCount++
		log.V(2).Info("Engine: Store failed", "domain", s.Domain().Name(), "error", err, "config", s.Original)
	}
}

// Get (re-)creates the store as required. Concurrent safe.
func (s *storeHolder) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	store, err := s.ensureLH()
	if err != nil {
		return err
	}
	func() { // Unlock around call to Get()
		s.lock.Unlock()
		defer s.lock.Lock()
		err = store.Get(ctx, q, constraint, result)
	}()
	if err != nil {
		s.recordErrorLH(err)
		if s.Store != nil && s.Original != nil {
			if c, _ := s.Store.(io.Closer); c != nil {
				_ = c.Close()
			}
			s.Store = nil
			s.retryAfter = time.Now().Add(s.Engine.Tuning.GetStoreRetryInterval())
		}
	}

	return err
}

// Ensure the store is connected. Concurrent safe.
func (s *storeHolder) Ensure() (korrel8r.Store, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	ks, err := s.ensureLH()
	return ks, err
}

// ensureLH is unsafe, must be called with lock held.
func (s *storeHolder) ensureLH() (_ korrel8r.Store, err error) {
	if s.Store != nil {
		return s.Store, nil
	}
	if s.Original == nil {
		return nil, fmt.Errorf("no store configuration for domain %v", s.domain.Name())
	}
	if time.Now().Before(s.retryAfter) {
		return nil, s.LastErr
	}
	defer func() {
		if err != nil {
			s.recordErrorLH(err)
			s.retryAfter = time.Now().Add(s.Engine.Tuning.GetStoreRetryInterval())
		}
	}()

	// Expand the store config each time - the results may change.
	s.Expanded = config.Store{}
	for k, original := range s.Original {
		var expanded string
		expanded, err = s.Engine.execTemplate(s.domain.Name()+"-store", original, nil)
		if err != nil {
			var execErr template.ExecError
			if errors.As(err, &execErr) {
				if err2 := errors.Unwrap(execErr.Err); err2 != nil {
					err = err2
				}
			}
			return nil, err
		}
		s.Expanded[k] = expanded
	}
	// Create the store
	if _, ok := s.Expanded[config.StoreKeyMock]; ok {
		s.Store, err = mock.NewStoreConfig(s.domain, s.Expanded)
	} else {
		s.Store, err = s.domain.Store(s.Expanded)
	}
	if err != nil {
		s.Store = nil
	}
	if err == nil {
		log.V(2).Info("Engine: Store connected", "domain", s.Domain().Name(), "config", s.Original)
	}
	return s.Store, err
}

// storeHolders contains multiple store wrappers storeHolders and iterates over them in Get.
type storeHolders struct {
	domain korrel8r.Domain
	stores []*storeHolder
}

func newStoreHolders(d korrel8r.Domain) *storeHolders {
	return &storeHolders{
		domain: d,
		stores: []*storeHolder{},
	}
}

func (ss *storeHolders) Domain() korrel8r.Domain { return ss.domain }

func (ss *storeHolders) Add(newStore *storeHolder) {
	// Check for duplicate configuration
	if newStore.Original != nil && slices.ContainsFunc(ss.stores,
		func(s *storeHolder) bool { return reflect.DeepEqual(s.Original, newStore.Original) }) {
		return // Ignore duplicates
	}
	ss.stores = append(ss.stores, newStore)
}

func (ss *storeHolders) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	errs := unique.NewList[string]()
	ok := false
	for _, s := range ss.stores {
		// Iterate over stores and accumulate all results.
		err := s.Get(ctx, q, constraint, result)
		if err != nil {
			errs.Add(err.Error())
		}
		ok = (err == nil) || ok // Remember if any call succeeds.
	}
	if ok { // If any call succeeded, this is a success
		if len(errs.List) > 0 {
			log.V(2).Info("Get succeeded with non-fatal errors", "errors", errs.List)
		}
		return nil
	}
	return fmt.Errorf("Get failed: %v", errs.List)
}

// Configs returns the expanded configurations for each store.
func (ss *storeHolders) Configs() (ret []config.Store) {
	for _, s := range ss.stores {
		sc := maps.Clone(s.Expanded)
		if s.LastErr != nil {
			sc[config.StoreKeyError] = s.LastErr.Error()
		}
		if s.ErrCount > 0 {
			sc[config.StoreKeyErrorCount] = strconv.Itoa(s.ErrCount)
		}
		ret = append(ret, sc)
	}
	return ret
}

// Ensure calls [configuredStore.Ensure] on all configured stores.
func (ss *storeHolders) Ensure() (ks []korrel8r.Store) {
	for _, s := range ss.stores {
		// Not an error if create fails, will be logged by the store wrapper.
		if k, err := s.Ensure(); err == nil && k != nil {
			ks = append(ks, k)
		}
	}
	return ks
}
