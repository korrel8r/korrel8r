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
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
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
	stores        map[korrel8r.Domain]*stores
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
	return nil, korrel8r.DomainNotFoundError{Domain: name}
}

// StoreFor returns the aggregated store for a domain, may be nil.
func (e *Engine) StoreFor(d korrel8r.Domain) korrel8r.Store {
	return e.stores[d]
}

// StoresFor returns the list of individual stores for a domain.
func (e *Engine) StoresFor(d korrel8r.Domain) []korrel8r.Store {
	if ss := e.stores[d]; ss != nil {
		return ss.Ensure()
	}
	return nil
}

// StoreConfigsFor returns the expanded store configurations and status.
func (e *Engine) StoreConfigsFor(d korrel8r.Domain) []config.Store {
	if ss, ok := e.stores[d]; ok {
		return ss.Configs()
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
		return nil, korrel8r.ClassNotFoundError{Class: class, Domain: d}
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

func (e *Engine) Rule(name string) korrel8r.Rule { return e.rulesByName[name] }

// Graph creates a new graph of the engine's rules.
func (e *Engine) Graph() *graph.Graph { return graph.NewData(e.Rules()...).FullGraph() }

// Get results for query from all stores for the query domain.
func (e *Engine) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) (err error) {
	constraint = constraint.Default()
	if timeout := constraint.GetTimeout(); timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	defer func() {
		if err != nil {
			log.V(2).Info("Get failed", "error", err, "query", query, "constraint", constraint)
			err = fmt.Errorf("Get failed: %v : %w", query, err)
		}
	}()
	ss := e.stores[query.Class().Domain()]
	if ss == nil {
		return korrel8r.StoreNotFoundError{Domain: query.Class().Domain()}
	}
	r := result
	if log.V(3).Enabled() { // Don't add overhead unless trace logging is enabled.
		start := time.Now() // Measure latency
		count := 0          // Count results
		r = korrel8r.FuncAppender(func(o korrel8r.Object) { result.Append(o); count++ })
		defer func() {
			if err == nil {
				log.V(3).Info("Get succeeded", "count", count, "latency", time.Since(start), "query", query, "constraint", constraint)
			}
		}()
	}
	return ss.Get(ctx, query, constraint, r)
}

// Follower creates a follower. Constraint can be nil.
func (e *Engine) Follower(ctx context.Context, c *korrel8r.Constraint) *Follower {
	return &Follower{Engine: e, Context: ctx, Constraint: c.Default(), rules: map[appliedRule]graph.Queries{}}
}

// Start populates the start node with objects and results of queries.
// Queries and objects must be of the same class as the node.
func (e *Engine) Start(ctx context.Context, start *graph.Node, objects []korrel8r.Object, queries []korrel8r.Query, constraint *korrel8r.Constraint) error {
	start.Result.Append(objects...)
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
func (e *Engine) GoalSearch(ctx context.Context, g *graph.Graph, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query, constraint *korrel8r.Constraint, goals []korrel8r.Class) (*graph.Graph, error) {
	if err := e.Start(ctx, g.NodeFor(start), objects, queries, constraint); err != nil {
		return nil, err
	}
	f := e.Follower(ctx, constraint)
	return g.Traverse(start, goals, f.Traverse)
}

// Neighbours generates a neighbourhood graph from starting objects and queries.
func (e *Engine) Neighbours(ctx context.Context, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query, constraint *korrel8r.Constraint, depth int) (*graph.Graph, error) {
	f := e.Follower(ctx, constraint)
	g := e.Graph()
	if err := e.Start(ctx, g.NodeFor(start), objects, queries, constraint); err != nil {
		return nil, err
	}
	return g.Neighbours(start, depth, f.Traverse)
}

// query implements the template function.
func (e *Engine) query(query string) ([]korrel8r.Object, error) {
	q, err := e.Query(query)
	if err != nil {
		return nil, err
	}
	results := korrel8r.NewResult(q.Class())
	err = e.Get(context.Background(), q, nil, results)
	return results.List(), err
}

// NewTemplate returns a template set up with options and funcs for this engine.
// See package documentation for more.
func (e *Engine) NewTemplate(name string) *template.Template {
	// missingkey defines behaviour on lookup of a non-existent map key.
	// Returning a zero is more intuitive for rules than the arbitrary default "<no value>".
	return template.New(name).Funcs(e.templateFuncs).Option("missingkey=error")
}

// execTemplate is a convenience to call NewTemplate, execute the template and stringify the result.
func (e *Engine) execTemplate(tmplString string, data any) (string, error) {
	tmpl, err := e.NewTemplate(tmplString).Parse(tmplString)
	if err != nil {
		return "", err
	}
	w := &bytes.Buffer{}
	if err := tmpl.Execute(w, data); err != nil {
		return "", err
	}
	return w.String(), nil
}
