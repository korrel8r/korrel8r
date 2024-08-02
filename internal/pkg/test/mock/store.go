// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"sigs.k8s.io/yaml"
)

// Store is a mock store with a map of queries to  result functions.
type Store struct {
	// ConstraintFunc (optional) returns true if object is accepted.
	ConstraintFunc func(*korrel8r.Constraint, korrel8r.Object) bool

	domain  korrel8r.Domain
	Queries QueryMap
}

// QueryMap keys are query strings, values are one of these types:
//
// - QueryFunc: store calls the function to get results.
// - []korrel8r.Object: store returns the array of results.
// - korrel8r.Object: store returns []korrel8r.Object{value}
type QueryMap map[string]any

type QueryFunc func(korrel8r.Query) []korrel8r.Object

func NewStore(d korrel8r.Domain) *Store { return &Store{domain: d, Queries: QueryMap{}} }

func NewStoreWith(d korrel8r.Domain, m QueryMap) *Store {
	return &Store{domain: d, Queries: m}
}

func NewStoreConfig(d korrel8r.Domain, cfg any) (*Store, error) {
	s := NewStore(d)
	if cfg == nil {
		return s, nil // Nothing to load
	}
	cs, err := impl.TypeAssert[config.Store](cfg)
	if err != nil {
		return nil, err
	}
	file := cs[config.StoreKeyMock]
	if file == "" {
		return s, nil
	}
	if err := s.LoadFile(file); err != nil {
		return nil, fmt.Errorf("failed to load %v=%q: %w", config.StoreKeyMock, file, err)
	}
	return s, nil
}

func (s *Store) Domain() korrel8r.Domain { return s.domain }

func (s *Store) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, r korrel8r.Appender) error {
	var result []korrel8r.Object
	data := s.Queries[q.String()]
	switch data := data.(type) {
	case nil:
		return nil
	case []korrel8r.Object:
		result = data
	case QueryFunc:
		result = data(q)
	default:
		result = []korrel8r.Object{data}
	}
	for i, o := range result {
		if limit := constraint.GetLimit(); limit > 0 && i >= limit {
			break
		}
		if s.ConstraintFunc == nil || constraint == nil || s.ConstraintFunc(constraint, o) {
			r.Append(o)
		}
	}
	return nil
}

func (s *Store) Resolve(korrel8r.Query) *url.URL { panic("not implemented") }

// Add queries and results
func (s *Store) Add(queries QueryMap) { maps.Copy(s.Queries, queries) }

// NewQuery returns a query that will get the result. The query data is the JSON string of the result.
func (s *Store) NewQuery(c korrel8r.Class, result ...korrel8r.Object) korrel8r.Query {
	b, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	q := NewQuery(c, string(b))
	s.Add(QueryMap{q.String(): result})
	return q
}

// LoadFile loads queries and results from a file.
func (s *Store) LoadFile(file string) error {
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return s.LoadData(b)
}

// LoadData loads queries and results from bytes..
func (s *Store) LoadData(data []byte) error {
	loaded := map[string][]json.RawMessage{}
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return err
	}
	for qs, raw := range loaded {
		q, err := s.Domain().Query(qs)
		if err != nil {
			return err
		}
		var result []korrel8r.Object
		for _, r := range raw {
			o, err := q.Class().Unmarshal([]byte(r))
			if err != nil {
				return err
			}
			result = append(result, o)
		}
		s.Add(QueryMap{q.String(): result})
	}
	return nil
}
