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
// Once created (see [Build]) an engine is immutable.
type Engine struct {
	domains       map[string]korrel8r.Domain
	stores        map[korrel8r.Domain]*stores
	templateFuncs template.FuncMap
	rulesByName   map[string]korrel8r.Rule
	rules         []korrel8r.Rule
}

func (e *Engine) Domains() []korrel8r.Domain {
	domains := maps.Values(e.domains)
	slices.SortFunc(domains, func(a, b korrel8r.Domain) int { return strings.Compare(a.Name(), b.Name()) })
	return domains
}

func (e *Engine) Domain(name string) (korrel8r.Domain, error) {
	if d := e.domains[name]; d != nil {
		return d, nil
	}
	return nil, korrel8r.DomainNotFoundError(name)
}

// StoreFor returns the aggregated store for a domain, nil if the domain does not exist.
func (e *Engine) StoreFor(d korrel8r.Domain) korrel8r.Store {
	s := e.stores[d]
	if s == nil {
		return nil
	}
	return s
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

// ClassesFor collects classes from  the domain or its stores.
func (e *Engine) ClassesFor(d korrel8r.Domain) ([]korrel8r.Class, error) {
	classes := d.Classes()
	if len(classes) > 0 {
		return classes, nil
	}
	if s := e.StoreFor(d); s != nil {
		return s.StoreClasses()
	}
	return nil, fmt.Errorf("no classes for domain: %v", d)
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
	d, err := e.Domain(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, korrel8r.ClassNotFoundError(class)
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
	ss := e.stores[query.Class().Domain()]
	if ss == nil {
		return fmt.Errorf("no stores found for domain %v", query.Class().Domain())
	}
	r := result
	if log.V(3).Enabled() { // Don't add overhead unless trace logging is enabled.
		start := time.Now() // Measure latency
		count := 0          // Count results
		r = korrel8r.AppenderFunc(func(o korrel8r.Object) { result.Append(o); count++ })
		defer func() {
			if err == nil {
				log.V(3).Info("Engine: Get OK", "n", count, "t", time.Since(start), "query", query)
			}
		}()
	}
	return ss.Get(ctx, query, constraint, r)
}

// query implements the template function 'query'.
func (e *Engine) query(query string) ([]korrel8r.Object, error) {
	q, err := e.Query(query)
	if err != nil {
		return nil, err
	}
	results := graph.NewResult(q.Class())
	err = e.Get(context.Background(), q, nil, results)
	return results.List(), err
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
