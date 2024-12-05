// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"fmt"
	"maps"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

type appliedRule struct {
	Rule  korrel8r.Rule
	Start korrel8r.Class
}

// syncTraverser provides Vist() and Traverse() methods to follow rules and collect results in a graph.
type syncTraverser struct {
	traverser
	ctx context.Context
	// temporary store for results of rules that need to be saved for a later line.
	rules map[appliedRule]graph.Queries
}

// NewSync returns a synchronous Traverser that evaluates queries sequentially.
func NewSync(e *engine.Engine, g *graph.Graph, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query) Traverser {
	return &syncTraverser{
		traverser: *newTraverser(e, g, start, objects, queries),
		rules:     map[appliedRule]graph.Queries{},
	}
}

func (t *syncTraverser) Goals(ctx context.Context, goals []korrel8r.Class) (*graph.Graph, error) {
	t.ctx = ctx
	if err := t.startNode(t.graph.NodeFor(t.start), t.objects, t.queries); err != nil {
		return nil, err
	}
	return t.graph.Traverse(t.start, goals, t)
}

func (t *syncTraverser) Neighbours(ctx context.Context, n int) (*graph.Graph, error) {
	t.ctx = ctx
	if err := t.startNode(t.graph.NodeFor(t.start), t.objects, t.queries); err != nil {
		return nil, err
	}
	return t.graph.Neighbours(t.start, n, t)
}

// Line a line gets all queries provided by Visit() on the From node,
// and stores results on the To node.
func (f *syncTraverser) Line(l *graph.Line) bool {
	rule := graph.RuleFor(l)
	start, goal := l.From().(*graph.Node), l.To().(*graph.Node)
	// Apply rule to each start object unless it was already applied to this start class.
	key := appliedRule{Start: start.Class, Rule: rule}
	if _, applied := f.rules[key]; !applied { // Not yet applied.
		f.rules[key] = graph.Queries{}
		for _, s := range start.Result.List() {
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
			l.Queries.Set(q, qc.Count) // Record on the count
			return true
		default: // Evaluate the query and store the results
			count, _ := f.getQuery(f.ctx, goal, q)
			l.Queries.Set(q, count)
			return true
		}
	})

	return len(l.Queries) > 0
}

func (f *syncTraverser) Node(*graph.Node) {}
func (f *syncTraverser) Edge(*graph.Edge) {}

// Start populates the start node with objects and results of queries.
// Queries and objects must be of the same class as the node.
func (f *syncTraverser) startNode(start *graph.Node, objects []korrel8r.Object, queries []korrel8r.Query) error {
	start.Result.Append(objects...)
	for _, query := range queries {
		if query.Class() != start.Class {
			return fmt.Errorf("class mismatch in query %v: expected class %v", query, start)
		}
		if _, err := f.getQuery(f.ctx, start, query); err != nil {
			return err
		}
	}
	return nil
}

func (f *syncTraverser) getQuery(ctx context.Context, goal *graph.Node, q korrel8r.Query) (int, error) {
	count := 0
	result := korrel8r.AppenderFunc(func(o korrel8r.Object) { goal.Result.Append(o); count++ })
	err := f.engine.Get(ctx, q, korrel8r.ConstraintFrom(f.ctx), result)
	goal.Queries.Set(q, count)
	return count, err
}
