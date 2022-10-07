// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/alanconway/korrel8/internal/pkg/logging"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/unique"
)

var log = logging.Log

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	Stores  map[string]korrel8.Store
	Domains map[string]korrel8.Domain
	Rules   *korrel8.RuleSet
}

func New() *Engine {
	return &Engine{Stores: map[string]korrel8.Store{}, Domains: map[string]korrel8.Domain{}, Rules: korrel8.NewRuleSet()}
}

func (e *Engine) ParseClass(name string) (korrel8.Class, error) {
	parts := strings.SplitN(name, "/", 2)
	domain := e.Domains[parts[0]]
	if domain == nil {
		return nil, fmt.Errorf("unknown domain: %q", parts[0])
	}
	var cname string
	if len(parts) == 2 {
		cname = parts[1]
	}
	class := domain.Class(cname)
	if class == nil {
		return nil, fmt.Errorf("unknown class: %q", parts[1])
	}
	return class, nil
}

// AddDomain domain and corresponding store, s may be nil.
func (e *Engine) AddDomain(d korrel8.Domain, s korrel8.Store) {
	e.Domains[d.String()] = d
	e.Stores[d.String()] = s
}

// Follow rules in a path.
func (e Engine) Follow(ctx context.Context, start korrel8.Object, c *korrel8.Constraint, path korrel8.Path) (queries []korrel8.Query, err error) {
	// TODO multi-path following needs thought, reduce duplication.
	if err := e.Validate(path); err != nil {
		return nil, err
	}
	starters := []korrel8.Object{start}
	for i, rule := range path {
		log.Info("following", "rule", rule)
		queries, err = e.followEach(rule, starters, c)
		if i == len(path)-1 || err != nil {
			break
		}
		d := rule.Goal().Domain()
		store := e.Stores[d.String()]
		if store == nil {
			return nil, fmt.Errorf("error following %v: no %v store", rule, d)
		}
		var result korrel8.ListResult
		for _, q := range queries {
			if err := store.Get(ctx, q, &result); err != nil {
				return nil, err
			}
		}
		starters = result.List()
	}

	return unique.InPlace(queries, unique.Same[korrel8.Query]), err
}

// Validate checks that the Goal() of each rule matches the Start() of the next,
// and that the engine has all the stores needed to follow the path.
func (e Engine) Validate(path korrel8.Path) error {
	for i, r := range path {
		if i < len(path)-1 {
			if r.Goal() != path[i+1].Start() {
				return fmt.Errorf("invalid path, mismatched rues: %v, %v", r, path[i+1])
			}
			d := r.Goal().Domain()
			if _, ok := e.Stores[d.String()]; !ok {
				return fmt.Errorf("no store available for %v", d)
			}
		}
	}
	return nil
}

// FollowEach calls r.Apply() for each start object and collects the resulting queries.
func (f Engine) followEach(rule korrel8.Rule, start []korrel8.Object, c *korrel8.Constraint) ([]korrel8.Query, error) {
	var queries []korrel8.Query
	for _, s := range start {
		q, err := rule.Apply(s, c)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	return unique.InPlace(queries, unique.Same[korrel8.Query]), nil
}
