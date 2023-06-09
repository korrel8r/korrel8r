// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Follower provide a Traverse() method to follow rules and collect results in a graph.
type Follower struct {
	Engine  *Engine
	Context context.Context
	Err     error // Collect errors using multierror
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
		query, err := rule.Apply(s, nil)
		if err != nil {
			log.V(4).Info("did not apply", "error", err)
			continue
		}
		log := log.WithValues("query", logging.JSON(query))
		result := korrel8r.NewCountResult(goalNode.Result)
		if err := f.Engine.Get(f.Context, goalNode.Class, query, result); err != nil {
			log.Error(err, "get failed")
		}
		l.QueryCounts.Put(query, result.Count)
		goalNode.QueryCounts.Put(query, result.Count)
		log.V(3).Info("results", "count", result.Count)
	}
}
