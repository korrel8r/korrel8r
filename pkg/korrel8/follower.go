package korrel8

import (
	"context"
	"fmt"
)

// Follower can follow a path of rules, looking up intermediate results as needed.
type Follower struct {
	Stores map[Domain]Store
}

// Follow rules in a path.
func (f Follower) Follow(ctx context.Context, start Object, c *Constraint, path Path) (result Queries, err error) {
	// FIXME multi-path following needs thought, reduce duplication.
	if err := f.Validate(path); err != nil {
		return nil, err
	}
	starters := []Object{start}
	for i, rule := range path {
		log.Info("following", "rule", rule)
		result, err = f.followEach(rule, starters, c)
		if i == len(path)-1 || err != nil {
			break
		}
		d := rule.Goal().Domain()
		store := f.Stores[d]
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
// and that the Follower has all the stores needed to follow the path.
func (f Follower) Validate(path Path) error {
	for i, r := range path {
		if i < len(path)-1 {
			if r.Goal() != path[i+1].Start() {
				return fmt.Errorf("invalid path, mismatched rues: %v, %v", r, path[i+1])
			}
			d := r.Goal().Domain()
			if _, ok := f.Stores[d]; !ok {
				return fmt.Errorf("no store available for %v", d)
			}
		}
	}
	return nil
}

// FollowEach calls r.Apply() for each start object and collects the resulting queries.
func (f Follower) followEach(r Rule, start []Object, c *Constraint) (Queries, error) {
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
