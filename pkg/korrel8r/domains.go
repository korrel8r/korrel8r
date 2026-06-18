// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"maps"
	"slices"
	"strings"
	"sync"
)

// Domains is a collection of [korrel8r.Domain].
// Concurrent safe.
type Domains struct {
	mu      sync.RWMutex
	domains map[string]Domain
	queries map[string]Query
}

func NewDomains() *Domains {
	return &Domains{
		domains: map[string]Domain{},
		queries: map[string]Query{},
	}
}

func (ds *Domains) Add(d Domain) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.domains[d.Name()] = d
}

// Get returns the domain with the given name, or nil if not found.
func (ds *Domains) Get(name string) Domain {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.domains[name]
}

func (ds *Domains) domain(name string) (Domain, error) {
	if d := ds.domains[name]; d != nil {
		return d, nil
	}
	return nil, NewDomainNotFoundError(name)
}

func (ds *Domains) Domain(name string) (Domain, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.domain(name)
}

func (ds *Domains) List() []Domain {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	ret := slices.Collect(maps.Values(ds.domains))
	slices.SortFunc(ret, func(a, b Domain) int { return strings.Compare(a.Name(), b.Name()) })
	return ret
}

func (ds *Domains) domainClass(domain, class string) (Class, error) {
	d, err := ds.domain(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, NewClassNotFoundError(domain, class)
	}
	return c, nil
}

func (ds *Domains) Class(fullname string) (Class, error) {
	d, c, err := ClassSplit(fullname)
	if err != nil {
		return nil, err
	}
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.domainClass(d, c)
}

func (ds *Domains) DomainClass(domain, class string) (Class, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.domainClass(domain, class)
}

// Query parses a query string.
// Queries are interned; equal strings will return identical Query instances.
func (ds *Domains) Query(query string) (Query, error) {
	if q := ds.rQuery(query); q != nil {
		return q, nil
	}
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if q := ds.queries[query]; q != nil {
		return q, nil
	}
	domain, _, _, err := QuerySplit(query)
	if err != nil {
		return nil, err
	}
	d, err := ds.domain(domain)
	if err != nil {
		return nil, err
	}
	q, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	ds.queries[query] = q
	return q, nil
}

func (ds *Domains) rQuery(query string) Query {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.queries[query]
}
