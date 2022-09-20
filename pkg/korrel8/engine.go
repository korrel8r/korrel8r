package korrel8

import (
	"context"
	"fmt"
	"strings"
)

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	Stores  map[string]Store
	Domains map[string]Domain
	Rules   *RuleSet
}

func NewEngine() *Engine {
	return &Engine{Stores: map[string]Store{}, Domains: map[string]Domain{}, Rules: NewRuleSet()}
}

func (e *Engine) ParseClass(name string) (Class, error) {
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

// Add domain and corresponding store, s may be nil.
func (e *Engine) Add(d Domain, s Store) {
	e.Domains[d.String()] = d
	e.Stores[d.String()] = s
}

// Follow rules in a path.
func (e Engine) Follow(ctx context.Context, start Object, c *Constraint, path Path) (result Queries, err error) {
	// TODO multi-path following needs thought, reduce duplication.
	if err := e.Validate(path); err != nil {
		return nil, err
	}
	starters := []Object{start}
	for i, rule := range path {
		log.Info("following", "rule", rule)
		result, err = e.followEach(rule, starters, c)
		if i == len(path)-1 || err != nil {
			break
		}
		d := rule.Goal().Domain()
		store := e.Stores[d.String()]
		if store == nil {
			return nil, fmt.Errorf("error following %v: no %v store", rule, d)
		}
		if starters, err = result.Get(ctx, store); err != nil {
			return nil, err
		}
		starters = uniqueObjectList(starters)
	}
	return result, err
}

// Validate checks that the Goal() of each rule matches the Start() of the next,
// and that the engine has all the stores needed to follow the path.
func (e Engine) Validate(path Path) error {
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
func (f Engine) followEach(r Rule, start []Object, c *Constraint) (Queries, error) {
	results := unique[string]{}
	for _, s := range start {
		result, err := r.Apply(s, c)
		if err != nil {
			return nil, err
		}
		results.add(result)
	}
	return results.list(), nil
}
