// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"golang.org/x/exp/maps"
)

var log = logging.Log()

// Engine manages a set of rules and stores to perform correlation.
// Once created (see [Build()]) an engine is immutable.
type Engine struct {
	domains       map[string]korrel8r.Domain
	stores        map[korrel8r.Domain][]korrel8r.Store
	storeConfigs  map[korrel8r.Domain][]korrel8r.StoreConfig
	templateFuncs template.FuncMap
	rulesByName   map[string]korrel8r.Rule
	rules         []korrel8r.Rule
}

// Domain returns the named domain or nil if not found.
func (e *Engine) Domain(name string) korrel8r.Domain { return e.domains[name] }

func (e *Engine) Domains() []korrel8r.Domain {
	domains := maps.Values(e.domains)
	slices.SortFunc(domains, func(a, b korrel8r.Domain) int { return strings.Compare(a.Name(), b.Name()) })
	return domains
}
func (e *Engine) DomainErr(name string) (korrel8r.Domain, error) {
	if d := e.Domain(name); d != nil {
		return d, nil
	}
	return nil, korrel8r.DomainNotFoundErr{Domain: name}
}

// StoresFor returns the known stores for a domain.
func (e *Engine) StoresFor(d korrel8r.Domain) []korrel8r.Store {
	return slices.Clone(e.stores[d])
}

// StoreConfigsFor returns store configurations added with AddStoreConfig
func (e *Engine) StoreConfigsFor(d korrel8r.Domain) []korrel8r.StoreConfig {
	return slices.Clone(e.storeConfigs[d])
}

// expandStoreConfig expands templates in store config values providing the engine's
func (e *Engine) expandStoreConfig(sc korrel8r.StoreConfig) error {
	for k, v := range sc {
		t, err := template.New(k + ": " + v).Funcs(e.TemplateFuncs()).Parse(v)
		if err != nil {
			return err
		}
		w := &bytes.Buffer{}
		err = t.Execute(w, nil)
		if err != nil {
			return err
		}
		sc[k] = w.String()
	}
	return nil
}

// Class parses a full class name and returns the
func (e *Engine) Class(fullname string) (korrel8r.Class, error) {
	d, c := impl.ClassSplit(fullname)
	if c == "" {
		return nil, fmt.Errorf("invalid class name: %v", fullname)
	} else {
		return e.DomainClass(d, c)
	}
}

func (e *Engine) DomainClass(domain, class string) (korrel8r.Class, error) {
	d, err := e.DomainErr(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, korrel8r.ClassNotFoundErr{Class: class, Domain: d}
	}
	return c, nil
}

// Query parses a query string to a query object.
func (e *Engine) Query(query string) (korrel8r.Query, error) {
	d, _, ok := strings.Cut(query, korrel8r.NameSeparator)
	if !ok {
		return nil, fmt.Errorf("invalid query string: %v", query)
	}
	domain, err := e.DomainErr(d)
	if err != nil {
		return nil, err
	}
	return domain.Query(query)
}

func (e *Engine) Rules() []korrel8r.Rule { return slices.Clone(e.rules) }

// Graph creates a new graph of the engine's rules.
func (e *Engine) Graph() *graph.Graph { return graph.NewData(e.Rules()...).FullGraph() }

// TemplateFuncs returns template helper functions for stores and domains known to this engine.
// See text/template.Template.Funcs
func (e *Engine) TemplateFuncs() map[string]any { return e.templateFuncs }

// Get results for query from all stores for the query domain.
func (e *Engine) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	for _, store := range e.stores[q.Class().Domain()] {
		if err := store.Get(ctx, q, constraint, result); err != nil {
			return err
		}
	}
	return nil
}

// Follower creates a follower. Constraint can be nil.
func (e *Engine) Follower(ctx context.Context, c *korrel8r.Constraint) *Follower {
	return &Follower{Engine: e, Context: ctx, Constraint: c, rules: map[appliedRule]graph.Queries{}}
}

// Start populates the start node for with objects and results of queries.
// Queries and objects must be of the same class as the node.
func (e *Engine) Start(ctx context.Context, start *graph.Node, objects []korrel8r.Object, queries []korrel8r.Query, constraint *korrel8r.Constraint) error {
	start.Result.Append(objects...) // FIXME verify objects are of correct class...
	for _, query := range queries {
		if query.Class() != start.Class {
			return fmt.Errorf("class mismatch in query %v: expected class %v", query, start)
		}
		count := 0
		counter := korrel8r.FuncAppender(func(o korrel8r.Object) { start.Result.Append(o); count++ })
		if err := e.Get(ctx, query, constraint, counter); err != nil {
			return err
		}
		start.Queries.Set(query, count)
	}
	return nil
}

// GoalSearch does a goal directed search from starting objects and queries, and returns the result graph.
func (e *Engine) GoalSearch(ctx context.Context, g *graph.Graph, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query, constraint *korrel8r.Constraint, goals []korrel8r.Class) error {
	if err := e.Start(ctx, g.NodeFor(start), objects, queries, constraint); err != nil {
		return err
	}
	f := e.Follower(ctx, constraint)
	g.Traverse(start, goals, f.Traverse)
	if f.Err != nil {
		return f.Err
	}
	return nil
}

// Neighbours generates a neighbourhood graph from starting objects and queries.
func (e *Engine) Neighbours(ctx context.Context, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query, constraint *korrel8r.Constraint, depth int) (*graph.Graph, error) {
	f := e.Follower(ctx, constraint)
	g := e.Graph()
	if err := e.Start(ctx, g.NodeFor(start), objects, queries, constraint); err != nil {
		return nil, err
	}
	g.Neighbours(start, depth, f.Traverse)
	if f.Err != nil {
		return nil, f.Err
	}
	return g, nil
}

// get implements the template function version of Get()
func (e *Engine) get(query string) ([]korrel8r.Object, error) {
	q, err := e.Query(query)
	if err != nil {
		return nil, err
	}
	results := korrel8r.NewResult(q.Class())
	err = e.Get(context.Background(), q, nil, results)
	return results.List(), err
}
