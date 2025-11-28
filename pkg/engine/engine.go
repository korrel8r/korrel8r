// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var log = logging.Log()

// Engine manages a set of rules and stores to perform correlation.
// Once created (see [Build]) an engine is immutable and concurrent safe.
type Engine struct {
	domains       korrel8r.Domains
	storeHolders  map[korrel8r.Domain]*storeHolders
	templateFuncs template.FuncMap
	rulesByName   map[string]korrel8r.Rule
	rules         []korrel8r.Rule

	// Tuning parameters
	Tuning config.Tuning
}

func (e *Engine) Domains() []korrel8r.Domain {
	domains := slices.Collect(maps.Values(e.domains))
	slices.SortFunc(domains, func(a, b korrel8r.Domain) int { return strings.Compare(a.Name(), b.Name()) })
	return domains
}

func (e *Engine) Domain(name string) (korrel8r.Domain, error) { return e.domains.Domain(name) }

// StoreFor returns the aggregated store for a domain, nil if there is none.
func (e *Engine) StoreFor(d korrel8r.Domain) korrel8r.Store {
	s := e.storeHolders[d]
	if s == nil || len(s.stores) == 0 {
		return nil
	}
	return s
}

// StoresFor returns the list of individual stores for a domain.
func (e *Engine) StoresFor(d korrel8r.Domain) []korrel8r.Store {
	if ss := e.storeHolders[d]; ss != nil {
		return ss.Ensure()
	}
	return nil
}

// StoreConfigsFor returns the expanded store configurations and status.
func (e *Engine) StoreConfigsFor(d korrel8r.Domain) []config.Store {
	if ss, ok := e.storeHolders[d]; ok {
		return ss.Configs()
	}
	return nil
}

// Class parses a full class name and returns the [korrel8r.Class]
func (e *Engine) Class(fullname string) (korrel8r.Class, error) {
	d, c := korrel8r.ClassSplit(fullname)
	if c == "" {
		return nil, fmt.Errorf("invalid class name: %v", fullname)
	} else {
		return e.DomainClass(d, c)
	}
}

func (e *Engine) Classes(fullnames []string) ([]korrel8r.Class, error) {
	var classes []korrel8r.Class
	for _, name := range fullnames {
		c, err := e.Class(name)
		if err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}
	return classes, nil
}

func (e *Engine) DomainClass(domain, class string) (korrel8r.Class, error) {
	d, err := e.Domain(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, fmt.Errorf("class not found: %v%v%v", domain, korrel8r.NameSeparator, class)
	}
	return c, nil
}

// Query parses a query string to a query object.
func (e *Engine) Query(query string) (korrel8r.Query, error) {
	query = strings.TrimSpace(query)
	d, _, ok := strings.Cut(query, korrel8r.NameSeparator)
	if !ok {
		return nil, fmt.Errorf("invalid query string: %v", query)
	}
	domain, err := e.Domain(d)
	if err != nil {
		return nil, err
	}
	return domain.Query(query)
}

// Queries parses a slice of query strings and returns a slice of query objects.
func (e *Engine) Queries(queryStrings []string) ([]korrel8r.Query, error) {
	var queries []korrel8r.Query
	for _, q := range queryStrings {
		query, err := e.Query(q)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}
	return queries, nil
}

func (e *Engine) Rules() []korrel8r.Rule { return slices.Clone(e.rules) }

func (e *Engine) Rule(name string) korrel8r.Rule { return e.rulesByName[name] }

// Graph creates a new graph of the engine's rules.
func (e *Engine) Graph() *graph.Graph { return graph.NewData(e.Rules()...).FullGraph() }

// Get results for query from all stores for the query domain.
func (e *Engine) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) (err error) {
	count := 0
	constraint = constraint.Default()

	defer func() {
		if err != nil {
			log.V(2).Info("Get failed", "error", err, "query", query.String(), "constraint", constraint.String())
		} else {
			log.V(5).Info("Get", "query", query.String(), "constraint", constraint.String(), "count", count)
		}
	}()
	ss := e.storeHolders[query.Class().Domain()]
	if len(ss.stores) == 0 {
		return fmt.Errorf("no stores found for domain %v", query.Class().Domain().Name())
	}
	return ss.Get(ctx, query, constraint, korrel8r.AppenderFunc(func(o korrel8r.Object) { count++; result.Append(o) }))
}

// NewTemplate returns a template set up with options and funcs for this engine.
// See package documentation for more.
func (e *Engine) NewTemplate(name string) *template.Template {
	// raise an error on lookup of a non-existent map key in a template.
	// Note use the sprig function "get" to return an empty value for missing keys.
	return template.New(name).Funcs(e.templateFuncs).Option("missingkey=error")
}

// execTemplate is a convenience to call NewTemplate, execute the template and stringify the result.
func (e *Engine) execTemplate(name, tmplString string, data any) (string, error) {
	tmpl, err := e.NewTemplate(name).Parse(tmplString)
	if err != nil {
		return "", err
	}
	w := &bytes.Buffer{}
	if err := tmpl.Execute(w, data); err != nil {
		return "", err
	}
	return w.String(), nil
}

// WithTimeout sets timeout on ctx, using configured default value if timeout == 0.
// If timeout == 0 and there is no default value, the returned context has no timeout.
func (e *Engine) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	if timeout == 0 && e.Tuning.RequestTimeout != nil {
		timeout = (*e.Tuning.RequestTimeout).Duration
	}
	if timeout == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
