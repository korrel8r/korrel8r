// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package traverse traverses graphs to find related objects.
package traverse

import (
	"context"
	"sync"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Goals traverses all paths from start objects to all goal classes.
func Goals(ctx context.Context, e *engine.Engine, start Start, goals []korrel8r.Class) (*graph.Graph, error) {
	log.V(2).Info("Goal directed search", "start", start, "goals", goals, "constraint", korrel8r.ConstraintFrom(ctx))
	traverser := newTraverser(e)
	return traverser.run(ctx, start, func(v graph.Visitor) { traverser.graph.GoalSearch(start.Class, goals, v) })
}

// Neighbours traverses to all neighbours of the start objects, traversing links up to the given depth.
func Neighbours(ctx context.Context, e *engine.Engine, start Start, depth int) (*graph.Graph, error) {
	log.V(2).Info("Neighbourhood search", "start", start, "depth", depth, "constraint", korrel8r.ConstraintFrom(ctx))
	traverser := newTraverser(e)
	return traverser.run(ctx, start, func(v graph.Visitor) { traverser.graph.Neighbours(start.Class, depth, v) })
}

// Start point information for graph traversal.
type Start struct {
	Class   korrel8r.Class    // Start class.
	Objects []korrel8r.Object // Start objects, must be of Start class.
	Queries []korrel8r.Query  // Queries for start objects, must be of Start class.
}

var log = logging.Log()

type traverser struct {
	engine  *engine.Engine
	graph   *graph.Graph
	workers map[korrel8r.Class]*worker
}

// A worker gets queries from a store, applies rules and sends new queries to other workers.
// Workers operate on disjoint data (node, class and store) and communicate via channels.
type worker struct {
	*traverser
	node          *graph.Node
	rules         *unique.List[korrel8r.Rule]          // Outgoing Rules.
	inbox, outbox *unique.DedupList[string, queryLine] // Incoming and outgoing queries
	processed     int                                  // Count of node.Result already processed
}

// queryLine is a query and the graph line associated with it.
type queryLine struct {
	Query korrel8r.Query
	Line  *graph.Line
}

func (ql queryLine) ID() string { return ql.Query.String() }

func newTraverser(e *engine.Engine) *traverser {
	return &traverser{
		engine:  e,
		graph:   e.Graph(),
		workers: map[korrel8r.Class]*worker{},
	}
}

// run starts the search and waits for it to complete.
func (t *traverser) run(ctx context.Context, start Start, traverse func(v graph.Visitor)) (*graph.Graph, error) {
	// Create workers for the relevant parts of the graph.
	traverse(graph.FuncVisitor{
		NodeF: func(n *graph.Node) {
			t.newWorker(n)
		},
		LineF: func(l *graph.Line) bool {
			t.newWorker(l.Start())
			t.newWorker(l.Goal())
			t.workers[l.Start().Class].rules.Add(l.Rule)
			return true
		},
	})

	// Prime the start worker
	w := t.workers[start.Class]
	w.node.Result.Append(start.Objects...)
	for _, q := range start.Queries {
		w.inbox.Add(queryLine{Query: q})
	}

	// Run workers until no work is left.
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Collect workers with work, remove from workers to avoid cycles.
		working := make([]*worker, 0, len(t.workers))
		for c, w := range t.workers {
			if w.HasWork() {
				working = append(working, w)
				delete(t.workers, c)
			}
		}
		if len(working) == 0 {
			break // No work to do
		}

		// Process inboxes concurrently, fill outboxes
		var busy sync.WaitGroup
		for _, w := range working {
			busy.Add(1)
			go func() { defer busy.Done(); w.Run(ctx) }()
		}
		busy.Wait() // Wait for worker.Run() goroutines to complete.

		// Redistribute outboxes to inboxes.
		for _, w := range working {
			for _, ql := range w.outbox.List {
				if next := t.workers[ql.Query.Class()]; next != nil {
					// Ignore goals that don't have workers - outside our search.
					next.inbox.Add(ql)
				}
			}
			w.outbox.Clear()
		}
	}

	// Remove empty nodes and lines from the graph.
	t.graph.EachLine(func(l *graph.Line) {
		if l.Queries.Total() == 0 {
			t.graph.RemoveLine(l.F.ID(), l.T.ID(), l.ID())
		}
	})
	t.graph.EachNode(func(n *graph.Node) {
		if len(n.Result.List()) == 0 {
			t.graph.RemoveNode(n.ID())
		}
	})
	return t.graph, nil
}

func (t *traverser) newWorker(n *graph.Node) {
	if t.workers[n.Class] == nil {
		t.workers[n.Class] = &worker{
			traverser: t,
			node:      n,
			rules:     unique.NewList[korrel8r.Rule](),
			inbox:     unique.NewDeduplicator(queryLine.ID).List(),
			outbox:    unique.NewDeduplicator(queryLine.ID).List(),
		}
	}
}

func (w *worker) HasWork() bool {
	return len(w.inbox.List) > 0 || len(w.node.Result.List()) > w.processed
}

// Run processes queries in inbox and stores result queries in outbox
func (w *worker) Run(ctx context.Context) {
	defer func() {
		w.inbox.Clear()
		w.processed = len(w.node.Result.List())
	}()
	// Process queries from inbox
	for _, ql := range w.inbox.List {
		if ctx.Err() != nil {
			return
		}
		before := len(w.node.Result.List())
		// Error is logged by engine.Get
		_ = w.engine.Get(ctx, ql.Query, korrel8r.ConstraintFrom(ctx), w.node.Result)
		result := w.node.Result.List()[before:]
		w.node.Queries.Set(ql.Query, len(result))
		if ql.Line != nil {
			ql.Line.Queries.Set(ql.Query, len(result))
		}
	}

	// Apply rules to un-processed results, generate queries in outbox.
	for _, o := range w.node.Result.List()[w.processed:] {
		for _, r := range w.rules.List {
			if ctx.Err() != nil {
				return
			}
			queries, err := r.Apply(o)
			if err != nil || len(queries) == 0 {
				log.V(4).Info("Rule did not apply", "rule", r.Name(), "start", w.node.Class, "error", err)
			} else {
				for _, q := range queries {
					if line := w.graph.FindLine(w.node.Class, q.Class(), r); line != nil {
						log.V(4).Info("Rule applied", "line", line, "query", q)
						w.outbox.Add(queryLine{Query: q, Line: line})
					}
				}
			}
		}
	}
}
