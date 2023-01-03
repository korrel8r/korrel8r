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
	if s, ok := e.stores[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("no store for domain: %v", name)
}

// AddDomain domain and corresponding store, store may be nil.
func (e *Engine) AddDomain(d korrel8.Domain, s korrel8.Store) {
	e.domains[d.String()] = d
	e.stores[d.String()] = s
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

// Follow rules in a multi-path, return all references at the end of the path.
func (e *Engine) Follow(ctx context.Context, starters []korrel8.Object, c *korrel8.Constraint, path graph.MultiPath) ([]uri.Reference, error) {
	log.V(1).Info("follow: starting", "path", path)
	if !path.Valid() {
		return nil, fmt.Errorf("invalid path: %v", path)
	}

	refs := unique.NewList[uri.Reference]()
	for i, links := range path {
		refs.List = refs.List[:0] // Clear previous references
		log.V(1).Info("follow: follow rule set", "rules", links, "start", links.Start(), "goal", links.Goal())
		for _, rule := range links {
			r := e.followEach(rule, starters, c)
			refs.Append(r...)
		}
		log.V(1).Info("follow: got references", "domain", links.Goal(), "references", refs.List)
		if i == len(path)-1 || len(refs.List) == 0 {
			break
		}
		d := links.Goal().Domain()
		store := e.stores[d.String()]
		if store == nil {
			return nil, fmt.Errorf("no store for domain %v", d)
		}
		var result korrel8.ListResult
		for _, ref := range refs.List {
			if err := store.Get(ctx, ref, &result); err != nil {
				log.V(1).Error(err, "get error in follow, skipping", "query", ref)
			}
		}
		starters = result.List()
		log.V(1).Info("follow: found objects", "count", len(starters))
	}
	return refs.List, nil
}

func (e *Engine) FollowAll(ctx context.Context, starters []korrel8.Object, c *korrel8.Constraint, paths []graph.MultiPath) ([]uri.Reference, error) {
	// TODO: can we optimize multi-multi-path following by using common results?
	refs := unique.NewList[uri.Reference]()
	for _, p := range paths {
		r, err := e.Follow(ctx, starters, nil, p)
		if err != nil {
			return nil, err
		}
		refs.Append(r...)
	}
	return refs.List, nil
}

// FollowEach calls r.Apply() for each start object and collects the resulting references.
// Ignores (but logs) rules that fail to apply.
func (f *Engine) followEach(rule korrel8.Rule, start []korrel8.Object, c *korrel8.Constraint) []uri.Reference {
	var (
		refs = unique.NewList[uri.Reference]()
		merr error
	)
	for _, s := range start {
		r, err := rule.Apply(s, c)
		logContext := func() {
			log.V(3).Info("follow: start context", "object", s, "constraint", c)
		}
		switch {
		case err != nil:
			log.V(1).Info("follow: error applying rule", "rule", rule, "error", err)
			logContext()
		case r == uri.Reference{}:
			log.V(1).Info("follow: rule returned empty query", "rule", rule)
			logContext()
		default:
			log.V(2).Info("follow: rule returned query", "rule", rule, "query", r)
			refs.Append(r)
		}
		merr = multierr.Append(merr, err)
	}
	return refs.List
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
