// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"github.com/korrel8/korrel8/pkg/uri"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
)

var log = logging.Log()

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	stores        map[string]korrel8.Store
	domains       map[string]korrel8.Domain
	rules         []korrel8.Rule
	classes       []korrel8.Class
	graph         *graph.Graph
	graphOnce     sync.Once
	templateFuncs map[string]any
}

func New() *Engine {
	return &Engine{
		stores:        map[string]korrel8.Store{},
		domains:       map[string]korrel8.Domain{},
		templateFuncs: map[string]any{},
	}
}

// Domain gets a named domain.
func (e *Engine) Domain(name string) (korrel8.Domain, error) {
	if d, ok := e.domains[name]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("domain not found: %v", name)
}

func (e *Engine) Domains() (domains []korrel8.Domain) { return maps.Values(e.domains) }

// Store for domain or nil if no store is available.
func (e *Engine) Store(name string) (korrel8.Store, error) {
	if s := e.stores[name]; s != nil {
		return s, nil
	}
	return nil, fmt.Errorf("no store for domain: %v", name)
}

// AddDomain domain and corresponding store, store may be nil.
func (e *Engine) AddDomain(d korrel8.Domain, s korrel8.Store) {
	e.domains[d.String()] = d
	if s != nil {
		e.stores[d.String()] = s
	}
	// Stores and Domains implement TemplateFuncser if they provide template helper functions
	// for use by rules.
	for _, v := range []any{d, s} {
		if tf, ok := v.(korrel8.TemplateFuncser); ok {
			maps.Copy(e.templateFuncs, tf.TemplateFuncs())
		}
	}
}

// ParseClass parses a full 'domain/class' name and returns the class.
func (e *Engine) ParseClass(name string) (korrel8.Class, error) {
	d, c, ok := strings.Cut(name, "/")
	if !ok || c == "" || d == "" {
		return nil, fmt.Errorf("invalid class name: %v", name)
	}
	domain, err := e.Domain(d)
	if err != nil {
		return nil, err
	}
	class := domain.Class(c)
	if class == nil {
		return nil, fmt.Errorf("unknown class in domain %v: %v", d, c)
	}
	return class, nil
}

func (e *Engine) Rules() []korrel8.Rule { return e.rules }

func (e *Engine) AddRule(r korrel8.Rule) error {
	e.rules = append(e.rules, r)
	return nil
}

func (e *Engine) AddRules(rules ...korrel8.Rule) error {
	for _, r := range rules {
		if err := e.AddRule(r); err != nil {
			return err
		}
	}
	return nil
}

// Follow rules in a multi-path, collect result.
// Collects errors using multierr.Append
func (e *Engine) Follow(ctx context.Context, starters []korrel8.Object, c *korrel8.Constraint, path graph.MultiPath, results *Results) error {
	if !path.Valid() {
		return fmt.Errorf("invalid path: %v", path)
	}
	log.V(1).Info("follow path", "path", path)
	var merr error
	refs := unique.NewList[uri.Reference]()
	for i, links := range path {
		refs.List = refs.List[:0] // Clear previous references
		log.V(1).Info("follow links", "links", links, "start", links.Start(), "goal", links.Goal())
		for _, rule := range links {
			merr = multierr.Append(merr, e.followEach(rule, starters, c, &refs))
		}
		results.Get(links.Goal()).References.Append(refs.List...)
		if i == len(path)-1 || len(refs.List) == 0 {
			break
		}
		var objects korrel8.ListResult
		merr = multierr.Append(merr, e.GetAll(ctx, links.Goal(), refs.List, &objects))
		starters = objects.List()
		results.Get(links.Goal()).Objects.Append(starters...)
		log.V(1).Info("follow got", "class", links.Goal(), "count", len(starters))
	}
	return merr
}

// GetAll gets objects from all refs into result.
// Collects errors using multierr.Append
func (e *Engine) GetAll(ctx context.Context, class korrel8.Class, refs []uri.Reference, result korrel8.Result) error {
	store, err := e.Store(class.Domain().String())
	if err != nil {
		return err
	}
	var merr error
	for _, ref := range refs {
		if err := store.Get(ctx, ref, result); err != nil {
			merr = multierr.Append(merr, err)
		}
	}
	return merr
}

// GetLast gets the objects for the last result in results
func (e *Engine) GetLast(ctx context.Context, results *Results) error {
	rl := results.List
	if len(rl) == 0 {
		return nil
	}
	result := &rl[len(rl)-1]
	return e.GetAll(ctx, result.Class, result.References.List, result.Objects)

}

// FollowAll collects results from following multiple paths.
// Collects errors using multierr.Append
func (e *Engine) FollowAll(ctx context.Context, starters []korrel8.Object, c *korrel8.Constraint, paths []graph.MultiPath, results *Results) error {
	var merr error
	log.V(2).Info("follow all", "paths", paths, "objects", len(starters))
	// TODO: can we optimize multiple paths using topological sorting?
	for _, p := range paths {
		merr = multierr.Append(merr, multierr.Append(merr, e.Follow(ctx, starters, nil, p, results)))
	}
	return merr
}

// followEach calls r.Apply() for each start object and collects the resulting references.
// Collects errors using multierr.Append
func (f *Engine) followEach(rule korrel8.Rule, start []korrel8.Object, c *korrel8.Constraint, refs *unique.List[uri.Reference]) error {
	var merr error
	for _, s := range start {
		ref, err := rule.Apply(s, c)
		switch {
		case err != nil:
			merr = multierr.Append(merr, fmt.Errorf("error following %v: %w", korrel8.RuleName(rule), err))
		case ref == uri.Empty: // Ignore
		default:
			refs.Append(ref)
		}
	}
	return merr
}

// Graph computes the rule graph from e.Rules and e.Classes on the first call.
// On subsequent calls it returns the same graph, it is not re-computed.
func (e *Engine) Graph() *graph.Graph {
	e.graphOnce.Do(func() {
		e.graph = graph.New("korrel8", e.rules, e.classes)
	})
	return e.graph
}

// TemplateFuncs returns template helper functions for stores and domains known to this engine.
// See text/template.Template.Funcs
func (e *Engine) TemplateFuncs() map[string]any { return e.templateFuncs }

// RefConverter checks if either the named domain or store is a RefConverter
func (e *Engine) RefConverter(domain string) (korrel8.RefConverter, error) {
	d, err := e.Domain(domain)
	if err != nil {
		return nil, err
	}
	if cvt, ok := d.(korrel8.RefConverter); ok {
		return cvt, nil
	}
	s, err := e.Store(domain)
	if err != nil {
		return nil, err
	}
	if cvt, ok := s.(korrel8.RefConverter); ok {
		return cvt, nil
	}
	return nil, fmt.Errorf("no reference converter for %v", domain)
}

// RefClasser checks if either the named domain or store is a RefClasser
func (e *Engine) RefClasser(domain string) (korrel8.RefClasser, error) {
	d, err := e.Domain(domain)
	if err != nil {
		return nil, err
	}
	if cvt, ok := d.(korrel8.RefClasser); ok {
		return cvt, nil
	}
	s, err := e.Store(domain)
	if err != nil {
		return nil, err
	}
	if cvt, ok := s.(korrel8.RefClasser); ok {
		return cvt, nil
	}
	return nil, fmt.Errorf("can't deduce reference class for %v", domain)
}
