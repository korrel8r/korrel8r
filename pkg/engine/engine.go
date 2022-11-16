// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
)

var log = logging.Log

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	name      string
	stores    map[string]korrel8.Store
	domains   map[string]korrel8.Domain
	rules     []korrel8.Rule
	classes   []korrel8.Class
	graph     *graph.Graph
	graphOnce sync.Once
}

func New(name string) *Engine {
	return &Engine{name: name, stores: map[string]korrel8.Store{}, domains: map[string]korrel8.Domain{}}
}

func (e Engine) Name() string { return e.name }

// Domain or nil if no domain of that name exists.
func (e *Engine) Domain(name string) korrel8.Domain { return e.domains[name] }

func (e *Engine) Domains() (domains []korrel8.Domain) { return maps.Values(e.domains) }

// Store for domain or nil if no store is available.
func (e *Engine) Store(d korrel8.Domain) korrel8.Store { return e.stores[d.String()] }

// AddDomain domain and corresponding store, store may be nil.
func (e *Engine) AddDomain(d korrel8.Domain, s korrel8.Store) {
	e.domains[d.String()] = d
	e.stores[d.String()] = s
}

// ParseClass parses a full 'domain/class' name and returns the class.
func (e *Engine) ParseClass(name string) (korrel8.Class, error) {
	d, c, ok := strings.Cut(name, "/")
	if !ok || c == "" || d == "" {
		return nil, fmt.Errorf("invalid class name: %v", name)
	}
	domain := e.Domain(d)
	if domain == nil {
		return nil, fmt.Errorf("unknown domain in class name: %v", name)
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
	/// FIXME validate rules while adding
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

// Follow rules in a multi-path, return all queries at the end of the path.
func (e Engine) Follow(ctx context.Context, starters []korrel8.Object, c *korrel8.Constraint, path graph.MultiPath) ([]korrel8.Query, error) {
	log.V(1).Info("follow: starting", "path", path)
	if !path.Valid() {
		return nil, fmt.Errorf("invalid path: %v", path)
	}
	queries := unique.NewList[url.URL]()
	for i, links := range path {
		queries.List = queries.List[:0] // Clear previous queries
		log.V(1).Info("follow: follow rule set", "rules", links, "start", links.Start(), "goal", links.Goal())
		for _, rule := range links {
			q := e.followEach(rule, starters, c)
			queries.Append(q...)
		}
		log.V(1).Info("follow: got queries", "domain", links.Goal(), "queries", logging.URLs(queries.List))
		if i == len(path)-1 || len(queries.List) == 0 {
			break
		}
		d := links.Goal().Domain()
		store := e.stores[d.String()]
		if store == nil {
			return nil, fmt.Errorf("no store for domain %v", d)
		}
		var result korrel8.ListResult
		for _, q := range queries.List {
			if err := store.Get(ctx, &q, &result); err != nil {
				log.V(1).Error(err, "get error in follow, skipping", "query", q)
			}
		}
		starters = result.List()
		log.V(1).Info("follow: found objects, continuing", "count", len(starters))
	}
	return queries.List, nil
}

func (e Engine) FollowAll(ctx context.Context, starters []korrel8.Object, c *korrel8.Constraint, paths []graph.MultiPath) ([]korrel8.Query, error) {
	// TODO: can we optimize multi-multi-path following by using common results?
	queries := unique.NewList[korrel8.Query]()
	for _, p := range paths {
		q, err := e.Follow(ctx, starters, nil, p)
		if err != nil {
			return nil, err
		}
		queries.Append(q...)
	}
	return queries.List, nil
}

// FollowEach calls r.Apply() for each start object and collects the resulting queries.
// Ignores (but logs) rules that fail to apply.
func (f Engine) followEach(rule korrel8.Rule, start []korrel8.Object, c *korrel8.Constraint) []korrel8.Query {
	var (
		queries = unique.NewList[korrel8.Query]()
		merr    error
	)
	for _, s := range start {
		q, err := rule.Apply(s, c)
		logContext := func() {
			log.V(3).Info("follow: start context", "object", s, "constraint", c)
		}
		switch {
		case err != nil:
			log.V(1).Info("follow: error applying rule", "rule", rule, "error", err)
			logContext()
		case q == nil || *q == url.URL{}:
			log.V(1).Info("follow: rule returned nil or empty query", "rule", rule)
			logContext()
		default:
			log.V(2).Info("follow: rule returned query", "rule", rule, "query", q)
			queries.Append(*q)
		}
		merr = multierr.Append(merr, err)
	}
	return queries.List
}

// Graph computes the rule graph from e.Rules and e.Classes on the first call.
// On subsequent calls it returns the same graph, it is not re-computed.
func (e Engine) Graph() *graph.Graph {
	e.graphOnce.Do(func() {
		e.graph = graph.New(e.name, e.rules, e.classes)
	})
	return e.graph
}
