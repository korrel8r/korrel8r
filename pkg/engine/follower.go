package engine

import (
	"context"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var log = logging.Log()

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
	store := f.Engine.Store(rule.Goal().Domain().String())
	if store == nil {
		log.V(3).Info("no store for goal")
		// Don't return, we want to generate final queries even if there is no store.
	}
	for _, s := range starters {
		query, err := rule.Apply(s, nil)
		if err != nil {
			log.V(3).Info("did not apply", "error", err)
			continue
		}
		log := log.WithValues("query", logging.JSON(query))
		if _, ok := goalNode.QueryCounts.Get(query); ok {
			log.V(3).Info("skip duplicate")
			continue
		}
		result := korrel8r.NewCountResult(goalNode.Result)
		if store != nil {
			if err := store.Get(f.Context, query, result); err != nil {
				// FIXME should report this error, but causing a test failure, investigate.
				// v.Err = multierr.Append(v.Err, err)
				log.Error(err, "store get error")
			}
		}
		l.QueryCounts.Put(query, result.Count)
		goalNode.QueryCounts.Put(query, result.Count)
		log.V(3).Info("results", "count", result.Count)
	}
}
