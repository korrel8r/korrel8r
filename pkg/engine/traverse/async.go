// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// async traverses using goroutines and channels to process store queries concurrently when possible.
type async struct {
	engine *engine.Engine
	graph  *graph.Graph // full graph provided initially.
	errs   *unique.List[string]
}

// NewAsync returns an asynchronous Traverser that can do multiple store queries concurrently.
func NewAsync(e *engine.Engine, g *graph.Graph) Traverser {
	return &async{engine: e, graph: g, errs: unique.NewList[string]()}
}

// Goals runs a goal-directed search.
// Results and Queries are filled in on graph.
func (a *async) Goals(ctx context.Context, start Start, goals []korrel8r.Class) (*graph.Graph, error) {
	log.V(2).Info("Goal directed search", "start", start, "goals", goals, "constraint", korrel8r.ConstraintFrom(ctx))
	traverse := func(v graph.Visitor) { a.graph.GoalSearch(start.Class, goals, v) }
	return a.run(ctx, start, traverse)
}

// Goals runs a neighbourhood.
// Results and Queries are filled in on graph.
func (a *async) Neighbours(ctx context.Context, start Start, depth int) (*graph.Graph, error) {
	log.V(2).Info("Neighbourhood search", "start", start, "depth", depth, "constraint", korrel8r.ConstraintFrom(ctx))
	traverse := func(v graph.Visitor) { a.graph.Neighbours(start.Class, depth, v) }
	return a.run(ctx, start, traverse)
}

func (a *async) run(ctx context.Context, start Start, traverse func(v graph.Visitor)) (g *graph.Graph, err error) {
	// Visit to set up nodes, channels, sending counts .
	// g collects the sub-graph that was traversed.
	g = a.graph.Data.EmptyGraph()
	traverse(graph.FuncVisitor{
		LineF: func(l *graph.Line) bool {
			a.ensureNode(g, l.Goal()).Sending() // Count each start->goal line as a potential sender.
			g.SetLine(l)
			return true
		},
		NodeF: func(n *graph.Node) { a.ensureNode(g, n) },
	})

	// MUST add start objects before starting goroutines.
	n, err := g.NodeForErr(start.Class)
	if err != nil {
		return nil, err
	}
	startNode := getNode(n)
	if startNode == nil {
		return g, fmt.Errorf(`invalid start class: %v`, start.Class)
	}
	startNode.Result.Append(start.Objects...)

	var busy sync.WaitGroup
	defer func() {
		busy.Wait() // wait for all goroutines.
	}()

	// Breadth-first traversal of sub-graph to start the goroutines.
	g.BreadthFirst(start.Class,
		graph.FuncVisitor{
			NodeF: func(n *graph.Node) {
				busy.Add(1) // Record on busy count
				go func() {
					defer busy.Done() // Notify done.
					// Decrement sending count set in setup.Line, channel closes at 0.
					defer g.EachLineFrom(n, func(l *graph.Line) { getNode(l.Goal()).Close() })
					getNode(n).Run(ctx)
				}()
			},
		},
		nil)

	// Send start queries to the start node
	startNode.Sending()     // Notify the start node that we are sending.
	defer startNode.Close() // Notify the start node we are done.
	for _, q := range start.Queries {
		startNode.queryChan <- lineQuery{Query: q}
	}

	// will return when all goroutines have called busy.Done.
	return g, nil
}

// ensureNode attaches an async.node to a graph.Node if not already present.
func (a *async) ensureNode(g *graph.Graph, n *graph.Node) *node {
	if n.Value == nil {
		n.Value = &node{
			Node:      n,
			engine:    a.engine,
			g:         g,
			queryChan: make(chan lineQuery, 1),
		}
		g.MergeNode(n)
	}
	return getNode(n)
}

// node holds the async traversal state for a single graph node.
type node struct {
	*graph.Node

	engine    *engine.Engine
	g         *graph.Graph
	queryChan chan lineQuery // Incoming queries.
	senders   atomic.Int64   // Count of senders to queryChan.
}

// lineQuery is a query and the line it arrived on.
type lineQuery struct {
	Line  *graph.Line
	Query korrel8r.Query
}

// getNode gets the async.node attached to a graph.Node
func getNode(n *graph.Node) *node { return n.Value.(*node) }

// Sending indicates there is another sender to this node.
func (n *node) Sending() { n.senders.Add(1) }

// Close informs the node that one of its senders is finished.
// When the last sender finishes, the channel is closed.
func (n *node) Close() int64 {
	senders := n.senders.Add(-1)
	if senders == 0 {
		close(n.queryChan) // Run() will exit when channel is cleared.
	} else if senders < 0 {
		panic("Done called too many times: " + n.Class.String())
	}
	return senders
}

// Run handles incoming queries for a single node.
//
// It updates it's own graph node and lines incoming to that node with query and count data.
// No need to sync updates since each Run goroutine operates on a distinct data set.
func (n *node) Run(ctx context.Context) {

	// Handle start-up objects first if there are any.
	for _, o := range n.Result.List() {
		n.applyRules(ctx, o)
	}

	// Now process the incoming query channel.
	for lq := range n.queryChan {
		if ctx.Err() != nil {
			// If context is cancelled, drain the channel but don't process anything.
			// If we don't drain the channel, sending goroutines will deadlock.
			continue
		}

		l, q := lq.Line, lq.Query
		if n.Queries.Has(q) {
			continue // Already processed this query.
		}
		before := len(n.Result.List())
		// Error is logged by engine.Get
		// Keep queries with errors or empty results to record that we tried, so we won't try again.
		_ = n.engine.Get(ctx, q, korrel8r.ConstraintFrom(ctx), n.Result)
		result := n.Result.List()[before:]
		for _, o := range result {
			n.applyRules(ctx, o)
		}
		n.Queries.Set(q, len(result))
		if l != nil { // Initial queries don't have a line
			l.Queries.Set(q, len(result))
		}
	}
	if ctx.Err() != nil {
		log.Error(ctx.Err(), "Cancelled")
	}
}

// applyRules generates outgoing queries for an object.
func (n *node) applyRules(ctx context.Context, o korrel8r.Object) {
	// Rules with multiple goal classes can appear on multiple outbound lines.
	// We don't know the actual goal class till the rule is applied.
	// Apply the rule the first time it is encountered, store results to send on the line for the query class.
	applied := map[korrel8r.Rule]struct {
		q   korrel8r.Query
		err error
	}{}
	n.g.EachLineFrom(n.Node, func(l *graph.Line) {
		if ctx.Err() != nil {
			return // Stop if the context was cancelled.
		}
		qe, ok := applied[l.Rule] // Already applied?
		if !ok {                  // No, apply now
			qe.q, qe.err = l.Rule.Apply(o)
			applied[l.Rule] = qe
			if qe.q == nil {
				log.V(4).Info("Rule did not apply", "rule", l.Rule.Name(), "class", n.Class, "object", korrel8r.GetID(n.Class, o), "error", qe.err)
				return
			}
			log.V(4).Info("Applied rule", "rule", l, "query", qe.q)
		}
		if qe.q != nil && qe.q.Class() == l.Goal().Class { // Send query
			getNode(l.Goal()).queryChan <- lineQuery{Query: qe.q, Line: l}
		}
	})
}
