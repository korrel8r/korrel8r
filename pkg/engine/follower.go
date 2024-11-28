// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"maps"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

type appliedRule struct {
	Rule  korrel8r.Rule
	Start korrel8r.Class
}

// Follower provides Vist() and Traverse() methods to follow rules and collect results in a graph.
type Follower struct {
	Engine     *Engine
	Context    context.Context
	Constraint *korrel8r.Constraint

	// temporary store for results of rules that need to be saved for a later line.
	rules map[appliedRule]graph.Queries
}

func NewFollower(e *Engine, ctx context.Context, c *korrel8r.Constraint) *Follower {
	return &Follower{
		Engine:     e,
		Context:    ctx,
		Constraint: c.Default(),
		rules:      map[appliedRule]graph.Queries{},
	}
}

// Line a line gets all queries provided by Visit() on the From node,
// and stores results on the To node.
func (f *Follower) Line(l *graph.Line) bool {
	rule := graph.RuleFor(l)
	start, goal := l.From().(*graph.Node), l.To().(*graph.Node)
	// Apply rule to each start object unless it was already applied to this start class.
	key := appliedRule{Start: start.Class, Rule: rule}
	count := 0
	if _, applied := f.rules[key]; !applied { // Not yet applied.
		f.rules[key] = graph.Queries{}
		for _, s := range start.Result.List() {
			count++
			q, err := rule.Apply(s)
			if q == nil { // Rule does  not apply
				log.V(4).Info("Rule apply error", "rule", rule.Name(), "error", err, "id", korrel8r.GetID(start.Class, s))
			} else {
				f.rules[key].Set(q, -1)
				log.V(4).Info("Rule apply", "rule", rule.Name(), "query", q, "id", korrel8r.GetID(start.Class, s))
			}
		}
	}

	// Process and remove queries that match this line's goal, leave the rest.
	maps.DeleteFunc(f.rules[key], func(s string, qc graph.QueryCount) bool {
		q := qc.Query
		switch {
		case q.Class() != goal.Class: // Wrong goal, leave it for another line.
			return false
		case goal.Queries.Has(q): // Already evaluated on goal node.
			l.Queries.Set(q, qc.Count) // Record on the link
			return true
		default: // Evaluate the query and store the results
			var count int
			result := korrel8r.AppenderFunc(func(o korrel8r.Object) { goal.Result.Append(o); count++ })
			_ = f.Engine.Get(f.Context, q, f.Constraint, result)
			l.Queries.Set(q, count)
			goal.Queries.Set(q, count)
			return true
		}
	})

	return l.Queries.Total() > 0
}

func (f *Follower) Node(*graph.Node) {}
func (f *Follower) Edge(*graph.Edge) {}
