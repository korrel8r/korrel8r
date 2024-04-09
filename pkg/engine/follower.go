// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"fmt"
	"slices"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Follower provides Vist() and Traverse() methods to follow rules and collect results in a graph.
type Follower struct {
	Engine     *Engine
	Context    context.Context
	Constraint *korrel8r.Constraint
	Err        error // Collect errors using multierror
}

func (f *Follower) Visit(node *graph.Node, eachLine graph.Lines) {}

// Traverse a line gets all queries provided by Visit() on the From node,
// and stores results on the To node.
func (f *Follower) Traverse(l *graph.Line) {
	rule := graph.RuleFor(l)
	log := log.WithValues("rule", fmt.Sprint(rule))
	star, goal := l.From().(*graph.Node), l.To().(*graph.Node)
	// Apply rule only if not already applied
	if _, ok := star.RulesApplied[rule]; !ok {
		star.RulesApplied[rule] = applyTo(rule, star)
	}
	// Remove queries for this line's goal, leave others to be evaluated by the matching line.
	star.RulesApplied[rule] = slices.DeleteFunc(star.RulesApplied[rule], func(q korrel8r.Query) bool {
		if q.Class() != goal.Class {
			return false // Wrong class, leave for the matching line.
		}
		log = log.WithValues("query", q.String())
		if !goal.Queries.Has(q) { // Not already evaluated for goal
			result := korrel8r.NewCountResult(goal.Result) // Store in goal, but count the contribution.
			if err := f.Engine.Get(f.Context, q, f.Constraint, result); err != nil {
				log.V(3).Info("get", "error", err)
			}
			l.Queries.Set(q, result.Count)
			goal.Queries.Set(q, result.Count) // TODO duplication
			log.V(3).Info("results", "count", result.Count)
		}
		return true
	})
}

func applyTo(rule korrel8r.Rule, node *graph.Node) []korrel8r.Query {
	queries := make([]korrel8r.Query, 0, len(node.Result.List()))
	for _, s := range node.Result.List() {
		if q, err := rule.Apply(s); err != nil || q == nil {
			log.V(3).Info("did not apply", "error", err)
		} else {
			queries = append(queries, q)
		}
	}
	return queries
}
