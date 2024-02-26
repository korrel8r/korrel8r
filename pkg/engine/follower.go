// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Follower provide a Traverse() method to follow rules and collect results in a graph.
type Follower struct {
	Engine     *Engine
	Context    context.Context
	Constraint *korrel8r.Constraint
	Err        error // Collect errors using multierror
}

func (f *Follower) Traverse(l *graph.Line) {
	rule := graph.RuleFor(l)
	log := log.WithValues("rule", korrel8r.RuleName(rule))
	startNode, goalNode := l.From().(*graph.Node), l.To().(*graph.Node)

	starters := startNode.Result.List()
	if len(starters) == 0 {
		return
	}
	for _, s := range starters {
		query, err := rule.Apply(s)
		if err != nil || query == nil {
			log.V(4).Info("did not apply", "error", err)
			continue
		}
		qs := korrel8r.QueryName(query)
		log := log.WithValues("query", qs)
		result := korrel8r.NewCountResult(goalNode.Result)
		if err := f.Engine.Get(f.Context, query, f.Constraint, result); err != nil {
			log.V(4).Info("error in get", "error", err)
		}
		// TODO get rid of duplication of query counts, simplify code?
		l.Queries[qs] = result.Count
		goalNode.Queries[qs] = result.Count
		log.V(3).Info("results", "count", result.Count)
	}
}
