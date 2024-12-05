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

// Sequential traverser, nothing done in parallel.
type seq struct {
	Engine *engine.Engine
	Graph  *graph.Graph

	ctx      context.Context
	subGraph *graph.Graph
	// temporary store for results of rules that need to be saved for a later line.
	rules map[appliedRule]graph.Queries
}

// NewSync returns a synchronous Traverser that evaluates queries sequentially.
func NewSync(e *engine.Engine, g *graph.Graph) Traverser {
	return &seq{Engine: e, Graph: g, subGraph: g.Data.EmptyGraph(), rules: map[appliedRule]graph.Queries{}}
}

func (t *seq) Goals(ctx context.Context, start Start, goals []korrel8r.Class) (*graph.Graph, error) {
	t.ctx = ctx
	log.V(4).Info("Sync goal search", "start", start.Class, "goals", goals)
	if err := t.startNode(t.Graph.NodeFor(start.Class), start.Objects, start.Queries); err != nil {
		return nil, err
	}
	t.Graph.GoalSearch(start.Class, goals, t)
	return t.subGraph, nil
}

func (t *seq) Neighbours(ctx context.Context, start Start, depth int) (*graph.Graph, error) {
	t.ctx = ctx
	log.V(4).Info("Sync neighbours search", "start", start, "depth", depth)
	if err := t.startNode(t.Graph.NodeFor(start.Class), start.Objects, start.Queries); err != nil {
		return nil, err
	}
	t.Graph.Neighbours(start.Class, depth, t)
	return t.subGraph, nil
}

// Line a line gets all queries provided by Visit() on the From node,
// and stores results on the To node.
func (t *seq) Line(l *graph.Line) bool {
	start, goal := l.From().(*graph.Node), l.To().(*graph.Node)
	// Apply rule to each start object unless it was already applied to this start class.
	key := appliedRule{Start: start.Class, Rule: l.Rule}
	if _, applied := t.rules[key]; !applied { // Not yet applied.
		t.rules[key] = graph.Queries{}
		for _, s := range start.Result.List() {
			q, err := l.Rule.Apply(s)
			if q == nil { // Rule does  not apply
				log.V(4).Info("Rule apply error", "rule", l.Rule.Name(), "error", err, "id", korrel8r.GetID(start.Class, s))
			} else {
				t.rules[key].Set(q, -1)
				log.V(4).Info("Rule apply", "rule", l.Rule.Name(), "query", q, "id", korrel8r.GetID(start.Class, s))
			}
		}
	}

	// Process and remove queries that match this line's goal, leave the rest.
	maps.DeleteFunc(t.rules[key], func(s string, qc graph.QueryCount) bool {
		q := qc.Query
		switch {
		case q.Class() != goal.Class: // Wrong goal, leave it for another line.
			return false
		case goal.Queries.Has(q): // Already evaluated on goal node.
			l.Queries.Set(q, qc.Count) // Record on the count
			return true
		default: // Evaluate the query and store the results
			count, _ := t.getQuery(t.ctx, goal, q)
			l.Queries.Set(q, count)
			return true
		}
	})
	if len(l.Queries) > 0 {
		t.subGraph.SetLine(l)
		return true
	}
	return false
}

func (t *seq) Node(n *graph.Node) { t.subGraph.MergeNode(n) }

// Start populates the start node with objects and results of queries.
// Queries and objects must be of the same class as the node.
func (t *seq) startNode(start *graph.Node, objects []korrel8r.Object, queries []korrel8r.Query) error {
	start.Result.Append(objects...)
	for _, query := range queries {
		if query.Class() != start.Class {
			return fmt.Errorf("class mismatch in query %v: expected class %v", query, start)
		}
		if _, err := t.getQuery(t.ctx, start, query); err != nil {
			return err
		}
	}
	return nil
}

func (t *seq) getQuery(ctx context.Context, goal *graph.Node, q korrel8r.Query) (int, error) {
	count := 0
	result := korrel8r.AppenderFunc(func(o korrel8r.Object) { goal.Result.Append(o); count++ })
	err := t.Engine.Get(ctx, q, korrel8r.ConstraintFrom(t.ctx), result)
	goal.Queries.Set(q, count)
	return count, err
}
