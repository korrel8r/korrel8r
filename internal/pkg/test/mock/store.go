// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	yaml "sigs.k8s.io/yaml"
)

// Store is a mock store where queries are resolved by:
// - a function to compute the result.
// - a YAML/JSON file mapping query strings to results.
// - a directory of files with names that are queries, containing JSON results.
type Store struct {
	// ConstraintFunc (optional) returns true if object is accepted.
	ConstraintFunc func(*korrel8r.Constraint, korrel8r.Object) bool
	classes        unique.Set[korrel8r.Class]
	lookup         []QueryFunc
	queries        *QueryMap

	domain korrel8r.Domain
}

func NewStore(d korrel8r.Domain, c ...korrel8r.Class) *Store {
	// Always include a query function for a mock.[Query] that contains its own result.
	containsResult := func(q korrel8r.Query) ([]korrel8r.Object, error) {
		if mq, ok := q.(Query); ok {
			return mq.result, mq.err
		}
		return nil, nil
	}
	qm := NewQueryMap()
	return &Store{
		domain:  d,
		classes: unique.NewSet(c...),
		queries: qm,
		lookup:  []QueryFunc{containsResult, qm.Get},
	}
}

// NewStoreConfig loads a store from the file indicated by cfg in StoreKeyMock
func NewStoreConfig(d korrel8r.Domain, cfg any) (*Store, error) {
	s := NewStore(d)
	if cfg == nil {
		return s, nil // Nothing to load
	}
	cs, err := typeAssert[config.Store](cfg)
	if err != nil {
		return nil, err
	}
	file := cs[config.StoreKeyMock]
	if file == "" {
		return s, nil // Not a mock store configuration
	}
	stat, err := os.Stat(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load %v=%q: %w", config.StoreKeyMock, file, err)
	}
	if stat.IsDir() {
		s.AddDir(file)
		return s, nil
	}
	if err := s.LoadFile(file); err != nil {
		return nil, fmt.Errorf("failed to load %v=%q: %w", config.StoreKeyMock, file, err)
	}
	return s, nil
}

func (s *Store) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, r korrel8r.Appender) error {
	for _, f := range s.lookup {
		result, err := f(q)
		if err != nil {
			return err
		}
		for i, o := range result {
			if limit := constraint.GetLimit(); limit > 0 && i >= limit {
				break
			}
			if s.ConstraintFunc == nil || constraint == nil || s.ConstraintFunc(constraint, o) {
				r.Append(o)
			}
		}
	}
	return nil
}

func (s *Store) Domain() korrel8r.Domain         { return s.domain }
func (s *Store) Resolve(korrel8r.Query) *url.URL { panic("not implemented") }

func (s *Store) AddLookup(lookup QueryFunc) { s.lookup = append(s.lookup, lookup) }

func (s *Store) AddDir(dir string) { s.AddLookup(QueryDir(dir).Get) }

// Add query with result.
// Query can be a korrel8r.Query or a string.
// Result can be:
// - QueryFunc: returns the same func.
// - []korrel8r.Object or nil: the result for this query.
// - korrel8r.Object: a single object, result is []korrel8r.Object{value}
func (s *Store) AddQuery(q any, result any) {
	switch q := q.(type) {
	case korrel8r.Query:
		s.queries.Put(q.String(), queryFunc(result))
	case string:
		s.queries.Put(q, queryFunc(result))
	default:
		panic(fmt.Errorf("mock.Store.AddQuery: bad query: (%T)%v", q, q))
	}
}

// NewQuery returns a query that will get the result. The query data is the JSON string of the result.
func (s *Store) NewQuery(c korrel8r.Class, result ...korrel8r.Object) korrel8r.Query {
	b, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	q := NewQuery(c, string(b))
	s.AddQuery(q, result)
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
	if err := yaml.UnmarshalStrict(data, &loaded); err != nil {
		return err
	}
	for qs, raw := range loaded {
		q, err := s.domain.Query(qs)
		if err != nil {
			return err
		}
		s.classes.Add(q.Class())
		var result []korrel8r.Object
		for _, r := range raw {
			o, err := q.Class().Unmarshal([]byte(r))
			if err != nil {
				return err
			}
			result = append(result, o)
		}
		s.AddQuery(q, result)
	}
	return nil
}

// QueryFunc evaluates a query and returns a result.
type QueryFunc func(korrel8r.Query) ([]korrel8r.Object, error)

// queryFunc returns a query function based on v, which can be:
// - QueryFunc: returns the same func.
// - []korrel8r.Object or nil: the result for this query.
// - korrel8r.Object: a single object, result is []korrel8r.Object{value}
// - error: result is an error.
func queryFunc(v any) QueryFunc {
	switch v := v.(type) {
	case QueryFunc:
		return v
	case nil:
		return func(korrel8r.Query) ([]korrel8r.Object, error) { return nil, nil }
	case []korrel8r.Object:
		return func(korrel8r.Query) ([]korrel8r.Object, error) { return v, nil }
	case error:
		return func(korrel8r.Query) ([]korrel8r.Object, error) { return nil, v }
	default:
		return func(korrel8r.Query) ([]korrel8r.Object, error) { return []korrel8r.Object{v}, nil }
	}
}

type QueryMap struct {
	lock sync.Mutex
	m    map[string]QueryFunc
}

func NewQueryMap() *QueryMap { return &QueryMap{m: map[string]QueryFunc{}} }

func (m *QueryMap) Get(q korrel8r.Query) ([]korrel8r.Object, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if f, ok := m.m[q.String()]; ok {
		return f(q)
	}
	return nil, nil
}

func (m *QueryMap) Put(q string, f QueryFunc) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.m[q] = f
}

// QueryDir is a directory of query files containing results in ndjson format.
//
// The file names are URL query-escaped strings, to prevent problems with
// unusual characters in file names.
type QueryDir string

func (s QueryDir) Get(q korrel8r.Query) ([]korrel8r.Object, error) {
	filename := url.QueryEscape(q.String())
	f, err := os.Open(filepath.Join(string(s), filename))
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		var result []korrel8r.Object
		d := json.NewDecoder(f)
		for {
			var o korrel8r.Object
			switch err := d.Decode(&o); err {
			case nil:
				result = append(result, o)
			case io.EOF:
				return result, nil
			default:
				return nil, err
			}
		}
	}
}

func typeAssert[T any](x any) (v T, err error) {
	v, ok := x.(T)
	if !ok {
		err = fmt.Errorf("wrong type: want %T, got (%T)(%#v)", v, x, x)
	}
	return v, err
}
