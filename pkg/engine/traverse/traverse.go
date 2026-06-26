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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Goals traverses all paths from start objects to all goal classes.
func Goals(ctx context.Context, e *engine.Engine, start Start, goals []korrel8r.Class) (*graph.Graph, error) {
	log.V(2).Info("Goal directed search", "start", start, "goals", goals, "constraint", start.Constraint)
	g, err := e.Graph().GoalPaths(start.Class, goals)
	if err != nil {
		return nil, err
	}
	g, err = newTraverser(e, g, start.Constraint).run(ctx, start, -1)
	g.RemoveEmptyGoalPaths(goals)
	return g, err
}

// Neighbors traverses to all neighbors of the start objects, traversing links up to the given depth.
func Neighbors(ctx context.Context, e *engine.Engine, start Start, depth int) (*graph.Graph, error) {
	log.V(2).Info("Neighbourhood search", "start", start, "depth", depth, "constraint", start.Constraint)
	g, err := e.Graph().Neighbors(start.Class, depth) // Reduce the graph.
	if err != nil {
		return nil, err
	}
	return newTraverser(e, g, start.Constraint).run(ctx, start, depth)
}

// Start point information for graph traversal.
type Start struct {
	Class      korrel8r.Class       // Start class.
	Objects    []korrel8r.Object    // Start objects, must be of Start class.
	Queries    []korrel8r.Query     // Queries for start objects, must be of Start class.
	Constraint *korrel8r.Constraint // Constraint to apply during the traversal.
}

var log = logging.Log()

type traverser struct {
	engine     *engine.Engine
	graph      *graph.Graph
	workers    map[korrel8r.Class]*worker
	constraint *korrel8r.Constraint
}

// A worker gets queries from a store, applies rules and sends new queries to other workers.
// Workers operate on disjoint data (node, class and store) and communicate via channels.
type worker struct {
	*traverser
	node          *graph.Node
	rules         unique.Set[korrel8r.Rule] // Outgoing Rules.
	inbox, outbox queryBox                  // Incoming and outgoing queries
	processed     int                       // Count of node.Result already processed
}

type queryBox map[korrel8r.Query]queryLine

func (qb queryBox) Add(ctx context.Context, ql queryLine) bool {
	if _, ok := qb[ql.Query]; ok {
		metricDuplicateQueries.Add(ctx, 1, ql.MetricAttributes())
		return false // Already present
	}
	qb[ql.Query] = ql
	return true
}

// queryLine is a query and the graph line associated with it.
type queryLine struct {
	Query korrel8r.Query
	Line  *graph.Line
}

func (ql queryLine) ID() korrel8r.Query { return ql.Query }

func (ql queryLine) MetricAttributes() metric.MeasurementOption {
	queryAttr := attribute.String("query", ql.Query.String())
	if ql.Line != nil {
		return metric.WithAttributes(queryAttr, attribute.String("line", ql.Line.String()))
	}
	return metric.WithAttributes(queryAttr)
}

func newTraverser(e *engine.Engine, g *graph.Graph, c *korrel8r.Constraint) *traverser {
	return &traverser{
		engine:     e,
		graph:      g,
		workers:    map[korrel8r.Class]*worker{},
		constraint: c,
	}
}

// run starts the search and waits for it to complete.
// If depth is >= 0 the search is limited to that depth. If depth < 0 there is no depth limit.
func (t *traverser) run(ctx context.Context, start Start, depth int) (*graph.Graph, error) {

	// Prime the start worker
	startNode, err := t.graph.NodeForErr(start.Class)
	if err != nil {
		return nil, err
	}
	w := t.newWorker(startNode)
	w.node.Result.Append(start.Objects...)
	for _, q := range start.Queries {
		ql := queryLine{Query: q}
		w.inbox.Add(ctx, ql)
	}

	// Create workers for start and goal of all rules in the graph.
	t.graph.EachLine(func(l *graph.Line) {
		t.newWorker(l.Start())
		t.newWorker(l.Goal())
		t.workers[l.Start().Class].rules.Add(l.Rule)
	})

	// Run until no work is left or we have reached the requested depth.
	// Depth < 0 means no depth limit.
	for i := 0; i <= depth || depth < 0; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Collect workers with work.
		working := make([]*worker, 0, len(t.workers))
		for _, w := range t.workers {
			if w.HasWork() {
				working = append(working, w)
			}
		}
		if len(working) == 0 {
			break // No work to do
		}

		// Process inboxes concurrently, fill outboxes
		var busy sync.WaitGroup
		for _, w := range working {
			busy.Go(func() { ; w.Run(ctx) })
		}
		busy.Wait() // Wait for worker.Run() goroutines to complete.

		// Redistribute outboxes to inboxes. Not concurrent, touches all workers.
		for _, w := range working {
			for _, ql := range w.outbox {
				if next := t.workers[ql.Query.Class()]; next != nil {
					next.inbox.Add(ctx, ql)
				}
			}
			clear(w.outbox)
		}
	}
	t.graph.RemoveEmpty()
	return t.graph, nil
}

func (t *traverser) newWorker(n *graph.Node) *worker {
	w := t.workers[n.Class]
	if w == nil {
		w = &worker{
			traverser: t,
			node:      n,
			rules:     unique.NewSet[korrel8r.Rule](),
			inbox:     queryBox{},
			outbox:    queryBox{},
		}
		t.workers[n.Class] = w
	}
	return w
}

func (w *worker) HasWork() bool {
	return len(w.inbox) > 0 || len(w.node.Result.List()) > w.processed
}

// Run processes queries in inbox, populates graph results, applies rules, and stores new queries in outbox.
func (w *worker) Run(ctx context.Context) {
	defer func() {
		clear(w.inbox)
		w.processed = len(w.node.Result.List())
	}()
	// Process queries from inbox, apply status rules per query.
	statusRules := w.engine.StatusRulesFor(w.node.Class)
	for _, ql := range w.inbox {
		if ctx.Err() != nil {
			return
		}
		before := len(w.node.Result.List())
		// Error is logged by engine.Get
		_ = w.engine.Get(ctx, ql.Query, w.constraint, w.node.Result)
		result := w.node.Result.List()[before:]
		metricQueries.Add(ctx, 1, ql.MetricAttributes())
		w.node.Queries.Set(ql.Query, len(result))
		if ql.Line != nil {
			ql.Line.Queries.Set(ql.Query, len(result))
		}
		statusCounts := map[string]int{}
		for _, o := range result {
			for _, r := range statusRules {
				statuses, _ := r.Apply(o)
				for _, s := range statuses {
					statusCounts[s]++
				}
			}
		}
		if len(statusCounts) > 0 {
			w.node.Queries.AddStatuses(ql.Query, statusCounts)
		}
	}

	// Apply correlation rules to un-processed results, generate queries in outbox.
	for _, o := range w.node.Result.List()[w.processed:] {
		for r := range w.rules {
			if ctx.Err() != nil {
				return
			}
			queries, err := r.Apply(o)
			log.V(4).Info("Rule applied", "name", r.Name(), "start", w.node.Class, "error", err, "queries", len(queries))
			metricRules.Add(ctx, 1, metric.WithAttributes(
				attribute.String("rule", r.Name()),
				attribute.String("start", w.node.Class.String()),
				attribute.Int("queries", len(queries))))
			for _, q := range queries {
				if line := w.graph.FindLine(w.node.Class, q.Class(), r); line != nil {
					log.V(5).Info("Add line", "line", line, "query", q)
					ql := queryLine{Query: q, Line: line}
					w.outbox.Add(ctx, ql)
				}
			}
		}
	}
}
