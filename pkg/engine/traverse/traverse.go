// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package traverse finds correlated objects by traversing a rule graph.
//
// The algorithm uses concurrent workers fed by a shared query channel:
//
//  1. The full rule graph is reduced to relevant paths (goal-directed or depth-limited neighborhood).
//  2. A node is created for each class, protected by a mutex for concurrent access.
//  3. A fixed-size worker pool reads queryLines from a shared channel:
//     a. Each worker executes a query via engine.Get to collect objects.
//     b. New objects are added to the target node, applying correlation rules immediately.
//     c. Resulting queries are deduplicated and sent back to the channel.
//  4. Traversal completes when all in-flight work is done (tracked by sync.WaitGroup).
//  5. Empty nodes and lines are pruned from the result graph.
package traverse

import (
	"context"
	"runtime"
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
	g, err = newTraverser(e, g, start.Constraint, -1).run(ctx, start)
	g.RemoveEmptyGoalPaths(goals)
	return g, err
}

// Neighbors traverses to all neighbors of the start objects, traversing links up to the given depth.
func Neighbors(ctx context.Context, e *engine.Engine, start Start, depth int) (*graph.Graph, error) {
	log.V(2).Info("Neighbourhood search", "start", start, "depth", depth, "constraint", start.Constraint)
	g, err := e.Graph().Neighbors(start.Class, depth)
	if err != nil {
		return nil, err
	}
	return newTraverser(e, g, start.Constraint, depth).run(ctx, start)
}

// Start point information for graph traversal.
type Start struct {
	Class      korrel8r.Class       // Start class.
	Objects    []korrel8r.Object    // Start objects, must be of Start class.
	Queries    []korrel8r.Query     // Queries for start objects, must be of Start class.
	Constraint *korrel8r.Constraint // Constraint to apply during the traversal.
}

var log = logging.Log()

// queryLine is a query, the graph line that generated it, and its traversal depth.
type queryLine struct {
	Query korrel8r.Query
	Line  *graph.Line
	depth int
}

func (ql queryLine) MetricAttributes() metric.MeasurementOption {
	queryAttr := attribute.String("query", ql.Query.String())
	if ql.Line != nil {
		return metric.WithAttributes(queryAttr, attribute.String("line", ql.Line.String()))
	}
	return metric.WithAttributes(queryAttr)
}

type lineKey struct {
	start, goal korrel8r.Class
	rule        korrel8r.Rule
}

// node wraps a graph.Node with a mutex for concurrent access.
type node struct {
	mu sync.Mutex
	*graph.Node
	processed int // count of Result objects already rule-applied
}

// workQueue is an unbounded, mutex-protected FIFO queue.
// put never blocks, so producer-consumer deadlock is impossible.
type workQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []queryLine
	closed bool
}

func newWorkQueue() *workQueue {
	q := &workQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *workQueue) put(ql queryLine) {
	q.mu.Lock()
	q.items = append(q.items, ql)
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *workQueue) get() (queryLine, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return queryLine{}, false
	}
	ql := q.items[0]
	q.items = q.items[1:]
	return ql, true
}

func (q *workQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Broadcast()
}

type traverser struct {
	engine     *engine.Engine
	graph      *graph.Graph
	constraint *korrel8r.Constraint
	maxDepth   int // -1 for unlimited

	// Read-only after init
	nodes     map[korrel8r.Class]*node
	rules     map[korrel8r.Class]unique.Set[korrel8r.Rule]
	lines     map[lineKey]*graph.Line
	ruleAttrs map[korrel8r.Rule]metric.MeasurementOption

	// Concurrent state
	work   *workQueue
	wg     sync.WaitGroup
	seenMu sync.Mutex
	seen   map[korrel8r.Query]struct{}
	lineMu sync.Mutex
}

func newTraverser(e *engine.Engine, g *graph.Graph, c *korrel8r.Constraint, maxDepth int) *traverser {
	t := &traverser{
		engine:     e,
		graph:      g,
		constraint: c,
		maxDepth:   maxDepth,
		nodes:      map[korrel8r.Class]*node{},
		rules:      map[korrel8r.Class]unique.Set[korrel8r.Rule]{},
		lines:      map[lineKey]*graph.Line{},
		ruleAttrs:  map[korrel8r.Rule]metric.MeasurementOption{},
		work:       newWorkQueue(),
		seen:       map[korrel8r.Query]struct{}{},
	}

	g.EachLine(func(l *graph.Line) {
		t.getOrCreateNode(l.Start())
		t.getOrCreateNode(l.Goal())
		startClass := l.Start().Class
		if t.rules[startClass] == nil {
			t.rules[startClass] = unique.NewSet[korrel8r.Rule]()
		}
		t.rules[startClass].Add(l.Rule)
		t.lines[lineKey{start: startClass, rule: l.Rule, goal: l.Goal().Class}] = l
		if _, ok := t.ruleAttrs[l.Rule]; !ok {
			t.ruleAttrs[l.Rule] = metric.WithAttributes(
				attribute.String("rule", l.Rule.Name()),
				attribute.String("start", startClass.String()))
		}
	})

	return t
}

func (t *traverser) getOrCreateNode(gn *graph.Node) *node {
	n := t.nodes[gn.Class]
	if n == nil {
		n = &node{Node: gn}
		t.nodes[gn.Class] = n
	}
	return n
}

// run launches the worker pool, primes start data, and waits for completion.
func (t *traverser) run(ctx context.Context, start Start) (*graph.Graph, error) {
	startGraphNode, err := t.graph.NodeForErr(start.Class)
	if err != nil {
		return nil, err
	}
	startNode := t.getOrCreateNode(startGraphNode)

	// Launch worker pool — workers block on the empty queue until work arrives.
	numWorkers := runtime.GOMAXPROCS(0)
	var workerWg sync.WaitGroup
	for range numWorkers {
		workerWg.Go(func() {
			for ql, ok := t.work.get(); ok; ql, ok = t.work.get() {
				t.handleQuery(ctx, ql)
			}
		})
	}

	// Sentinel prevents premature WaitGroup completion during priming.
	t.wg.Add(1)

	startNode.mu.Lock()
	startNode.Result.Append(start.Objects...)
	startNode.mu.Unlock()

	for _, q := range start.Queries {
		t.dedupAndSend(ctx, queryLine{Query: q, depth: 0})
	}

	t.applyRules(ctx, startNode, 1)

	t.wg.Done() // Release sentinel.
	t.wg.Wait()
	t.work.close()
	workerWg.Wait()

	t.graph.RemoveEmpty()
	return t.graph, ctx.Err()
}

// dedupAndSend checks depth and query dedup, then adds to the work queue.
func (t *traverser) dedupAndSend(ctx context.Context, ql queryLine) {
	if t.maxDepth >= 0 && ql.depth > t.maxDepth {
		return
	}
	if ctx.Err() != nil {
		return
	}
	if t.isDuplicate(ctx, ql) {
		return
	}
	t.wg.Add(1)
	t.work.put(ql)
}

func (t *traverser) isDuplicate(ctx context.Context, ql queryLine) bool {
	t.seenMu.Lock()
	defer t.seenMu.Unlock()
	if _, exists := t.seen[ql.Query]; exists {
		metricDuplicateQueries.Add(ctx, 1, ql.MetricAttributes())
		return true
	}
	t.seen[ql.Query] = struct{}{}
	return false
}

// handleQuery processes a single queryLine: executes the query, adds results, applies rules.
func (t *traverser) handleQuery(ctx context.Context, ql queryLine) {
	defer t.wg.Done()
	if ctx.Err() != nil {
		return
	}

	goalClass := ql.Query.Class()
	n := t.nodes[goalClass]
	if n == nil {
		return
	}
	if n.overLimit(t.constraint.GetQueryLimit()) {
		return
	}

	// Execute query into a local slice.
	var results []korrel8r.Object
	_ = t.engine.Get(ctx, ql.Query, t.constraint, korrel8r.AppenderFunc(func(objects ...korrel8r.Object) {
		results = append(results, objects...)
	}))
	metricQueries.Add(ctx, 1, ql.MetricAttributes())

	// Add unique new objects to node and record query.
	// The captured resultList slice header is safe to read after unlock because:
	// 1. Result only appends — elements at indices [before:len] are never modified.
	// 2. A concurrent append may grow the backing array, but the old array stays valid.
	// 3. We only read indices < our captured len, so concurrent writes at higher indices don't matter.
	n.mu.Lock()
	before := len(n.Result.List())
	for _, o := range results {
		n.Result.Add(o)
	}
	resultList := n.Result.List()
	resultCount := len(resultList) - before
	n.Queries.Set(ql.Query, resultCount)
	n.mu.Unlock()

	if ql.Line != nil {
		t.lineMu.Lock()
		ql.Line.Queries.Set(ql.Query, resultCount)
		t.lineMu.Unlock()
	}

	// Apply status rules to unique new objects.
	statusRules := t.engine.StatusRulesFor(goalClass)
	if len(statusRules) > 0 {
		statusCounts := map[string]int{}
		for _, o := range resultList[before:] {
			for _, r := range statusRules {
				statuses, _ := r.Apply(o)
				for _, s := range statuses {
					statusCounts[s]++
				}
			}
		}
		if len(statusCounts) > 0 {
			n.mu.Lock()
			n.Queries.AddStatuses(ql.Query, statusCounts)
			n.mu.Unlock()
		}
	}

	t.applyRules(ctx, n, ql.depth+1)
}

func (n *node) overLimit(limit int) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if limit > 0 && len(n.Queries) > limit {
		log.V(5).Info("Query limit reached", "class", n.Class, "queries", len(n.Queries))
		return true
	}
	return false
}

// applyRules applies outgoing correlation rules to unprocessed objects in a node.
// The processed counter ensures each object is rule-applied exactly once,
// even when multiple goroutines call this concurrently for the same node.
func (t *traverser) applyRules(ctx context.Context, n *node, nextDepth int) {
	// Snapshot the objects, update processed, release the lock
	n.mu.Lock()
	objects := n.Result.List()
	start := n.processed
	n.processed = len(objects)
	class := n.Class
	n.mu.Unlock()

	if start >= len(objects) {
		return
	}

	rules := t.rules[class]
	for _, o := range objects[start:] {
		for r := range rules {
			if ctx.Err() != nil {
				return
			}
			queries, err := r.Apply(o)
			log.V(4).Info("Rule applied", "name", r.Name(), "start", class, "error", err, "queries", len(queries))
			metricRules.Add(ctx, 1, t.ruleAttrs[r])
			for _, q := range queries {
				if line := t.lines[lineKey{start: class, rule: r, goal: q.Class()}]; line != nil {
					t.dedupAndSend(ctx, queryLine{Query: q, Line: line, depth: nextDepth})
				}
			}
		}
	}
}
