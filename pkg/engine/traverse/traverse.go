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
//  5. A result graph is built from only the nodes and lines that produced results.
package traverse

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"go.opentelemetry.io/otel/metric"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
)

// Goals traverses all paths from start objects to all goal classes.
func Goals(ctx context.Context, e *engine.Engine, start Start, goals []korrel8r.Class) (*graph.Graph, error) {
	log.V(2).Info("Goal directed search", "start", start, "goals", goals, "constraint", start.Constraint)
	shared := e.Graph()
	scope, err := goalScope(shared, start.Class, goals)
	if err != nil {
		return nil, err
	}
	g, err := newTraverser(e, shared.Data, scope, start.Constraint, -1).run(ctx, start)
	if err != nil {
		return nil, err
	}
	// Remove dead-end paths that don't reach a goal.
	g.RemoveEmptyGoalPaths(goals)
	return g, nil
}

// Neighbors traverses to all neighbors of the start objects, traversing links up to the given depth.
func Neighbors(ctx context.Context, e *engine.Engine, start Start, depth int) (*graph.Graph, error) {
	log.V(2).Info("Neighbourhood search", "start", start, "depth", depth, "constraint", start.Constraint)
	data := e.GraphData()
	scope, err := neighborScope(data, start.Class, depth)
	if err != nil {
		return nil, err
	}
	return newTraverser(e, data, scope, start.Constraint, depth).run(ctx, start)
}

// neighborScope returns the lines reachable within maxDepth BFS hops from start.
func neighborScope(data *graph.Data, start korrel8r.Class, maxDepth int) ([]*graph.Line, error) {
	u := data.NodeFor(start)
	if u == nil {
		return nil, fmt.Errorf("class not found in graph: %v", start)
	}

	nodeDepth := map[int64]int{u.ID(): 0}
	queue := []int64{u.ID()}
	var lines []*graph.Line
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		depth := nodeDepth[id]
		if depth >= maxDepth {
			continue
		}
		data.EachLineFromID(id, func(l *graph.Line) {
			lines = append(lines, l)
			goalID := l.To().ID()
			if _, seen := nodeDepth[goalID]; !seen {
				nodeDepth[goalID] = depth + 1
				queue = append(queue, goalID)
			}
		})
	}

	// Exclude lines that point toward a node at a shallower BFS depth.
	kept := lines[:0]
	for _, l := range lines {
		if nodeDepth[l.From().ID()] <= nodeDepth[l.To().ID()] {
			kept = append(kept, l)
		}
	}
	return kept, nil
}

// goalScope returns the lines on shortest/near-shortest paths from start to each goal.
func goalScope(shared *graph.Graph, start korrel8r.Class, goals []korrel8r.Class) ([]*graph.Line, error) {
	u, err := shared.NodeForErr(start)
	if err != nil {
		return nil, err
	}
	var lines []*graph.Line
	for _, goal := range goals {
		v, err := shared.NodeForErr(goal)
		if err != nil {
			return nil, err
		}
		paths := path.YenKShortestPaths(shared, math.MaxInt, 1, u, v)
		for _, p := range paths {
			for i := 1; i < len(p); i++ {
				ls := shared.Lines(p[i-1].ID(), p[i].ID())
				for ls.Next() {
					lines = append(lines, ls.Line().(*graph.Line))
				}
			}
		}
	}
	return lines, nil
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
	Query     korrel8r.Query
	lineIndex int // index into traverser.lineStates; -1 for a start query
	depth     int
}

type lineKey struct {
	start, goal int64
	rule        korrel8r.Rule
}

type lineState struct {
	line    *graph.Line
	queries graph.Queries // Allocated lazily when the line produces a query result.
}

// node holds mutable per-class state for the traversal overlay.
// nodeStatic is immutable topology and routing metadata, indexed by graph node ID.
type nodeStatic struct {
	class       korrel8r.Class
	rules       []korrel8r.Rule
	classMetric metric.MeasurementOption
}

// nodeState is mutable traversal state, allocated only when a node becomes active.
type nodeState struct {
	mu        sync.Mutex
	result    result.Result
	queries   graph.Queries
	processed int // count of result objects already rule-applied
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
	data       *graph.Data
	constraint *korrel8r.Constraint
	maxDepth   int // -1 for unlimited

	// Read-only after init
	nodeStatic []nodeStatic // Immutable routing, indexed by graph node ID.
	lineIndex  map[lineKey]int
	lineStates []lineState

	// Concurrent state
	nodeMu    sync.Mutex
	nodeState []*nodeState // Lazy mutable overlays, indexed by graph node ID.
	work      *workQueue
	wg        sync.WaitGroup
	seenMu    sync.Mutex
	seen      map[korrel8r.Query]struct{}
	lineMu    sync.Mutex
}

func newTraverser(e *engine.Engine, data *graph.Data, scopeLines []*graph.Line, c *korrel8r.Constraint, maxDepth int) *traverser {
	t := &traverser{
		engine:     e,
		data:       data,
		constraint: c,
		maxDepth:   maxDepth,
		nodeStatic: make([]nodeStatic, len(data.Nodes)),
		nodeState:  make([]*nodeState, len(data.Nodes)),
		lineIndex:  make(map[lineKey]int, len(scopeLines)),
		lineStates: make([]lineState, 0, len(scopeLines)),
		work:       newWorkQueue(),
		seen:       map[korrel8r.Query]struct{}{},
	}

	for _, l := range scopeLines {
		startID, goalID := l.From().ID(), l.To().ID()
		t.initNodeStatic(startID, l.Start().Class)
		t.initNodeStatic(goalID, l.Goal().Class)
		start := &t.nodeStatic[startID]
		if !slices.Contains(start.rules, l.Rule) {
			start.rules = append(start.rules, l.Rule)
		}
		key := lineKey{start: startID, rule: l.Rule, goal: goalID}
		if index, ok := t.lineIndex[key]; ok {
			t.lineStates[index].line = l
		} else {
			t.lineIndex[key] = len(t.lineStates)
			t.lineStates = append(t.lineStates, lineState{line: l})
		}
	}

	return t
}

func (t *traverser) initNodeStatic(id int64, class korrel8r.Class) {
	n := &t.nodeStatic[id]
	if n.class == nil {
		n.class = class
		n.classMetric = t.engine.ClassMetricAttrs(class)
	}
}

func (t *traverser) getOrCreateNodeState(id int64) *nodeState {
	t.nodeMu.Lock()
	defer t.nodeMu.Unlock()
	n := t.nodeState[id]
	if n == nil {
		n = &nodeState{
			result:  result.New(t.nodeStatic[id].class),
			queries: graph.Queries{},
		}
		t.nodeState[id] = n
	}
	return n
}

// run launches the worker pool, primes start data, and waits for completion.
func (t *traverser) run(ctx context.Context, start Start) (*graph.Graph, error) {
	startDataNode := t.data.NodeFor(start.Class)
	t.initNodeStatic(startDataNode.ID(), start.Class)
	startNode := t.getOrCreateNodeState(startDataNode.ID())

	// Launch worker pool — workers block on the empty queue until work arrives.
	numWorkers := runtime.GOMAXPROCS(0)
	var workerWg sync.WaitGroup
	for range numWorkers {
		workerWg.Go(func() {
			for ql, ok := t.work.get(); ok; ql, ok = t.work.get() {
				t.handleQuery(ctx, &ql)
			}
		})
	}

	// Sentinel prevents premature WaitGroup completion during priming.
	t.wg.Add(1)

	startNode.mu.Lock()
	startNode.result.Append(start.Objects...)
	startNode.mu.Unlock()

	for _, q := range start.Queries {
		t.dedupAndSend(ctx, queryLine{Query: q, lineIndex: -1, depth: 0})
	}

	t.applyRules(ctx, startDataNode.ID(), startNode, 1)

	t.wg.Done() // Release sentinel.
	t.wg.Wait()
	t.work.close()
	workerWg.Wait()

	return t.buildGraph(), ctx.Err()
}

// buildGraph creates a result graph containing only nodes and lines that produced results.
func (t *traverser) buildGraph() *graph.Graph {
	g := graph.New(t.data)
	nodeMap := make([]*graph.Node, len(t.nodeState))
	for id, n := range t.nodeState {
		if n == nil || len(n.result.List()) == 0 {
			continue
		}
		dn := t.data.Nodes[id]
		gn := &graph.Node{
			Node:    dn.Node,
			Class:   t.nodeStatic[id].class,
			Attrs:   graph.Attrs{},
			Result:  n.result,
			Queries: n.queries,
		}
		g.AddNode(gn)
		nodeMap[id] = gn
	}
	for _, state := range t.lineStates {
		if state.queries.Total() == 0 {
			continue
		}
		from, to := nodeMap[state.line.From().ID()], nodeMap[state.line.To().ID()]
		if from == nil || to == nil {
			continue
		}
		l := &graph.Line{
			Line:    multi.Line{F: from, T: to, UID: state.line.UID},
			Rule:    state.line.Rule,
			Attrs:   graph.Attrs{},
			Queries: state.queries,
		}
		g.AddLine(l)
	}
	return g
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
		id := t.data.NodeFor(ql.Query.Class()).ID()
		metricDuplicateQueries.Add(ctx, 1, t.nodeStatic[id].classMetric)
		return true
	}
	t.seen[ql.Query] = struct{}{}
	return false
}

// handleQuery processes a single queryLine: executes the query, adds results, applies rules.
func (t *traverser) handleQuery(ctx context.Context, ql *queryLine) {
	defer t.wg.Done()
	if ctx.Err() != nil {
		return
	}

	goalClass := ql.Query.Class()
	goalID := t.data.NodeFor(goalClass).ID()
	n := t.getOrCreateNodeState(goalID)
	if n == nil {
		return
	}
	if n.overLimit(t.constraint.GetQueryLimit(), goalClass) {
		return
	}

	// Execute query into a local slice.
	var results []korrel8r.Object
	_ = t.engine.Get(ctx, ql.Query, t.constraint, korrel8r.AppenderFunc(func(objects ...korrel8r.Object) {
		results = append(results, objects...)
	}))
	metricQueries.Add(ctx, 1, t.nodeStatic[goalID].classMetric)

	// Add unique new objects to node and record query.
	// The captured resultList slice header is safe to read after unlock because:
	// 1. Result only appends — elements at indices [before:len] are never modified.
	// 2. A concurrent append may grow the backing array, but the old array stays valid.
	// 3. We only read indices < our captured len, so concurrent writes at higher indices don't matter.
	n.mu.Lock()
	before := len(n.result.List())
	for _, o := range results {
		n.result.Add(o)
	}
	resultList := n.result.List()
	resultCount := len(resultList) - before
	n.queries.Set(ql.Query, resultCount)
	n.mu.Unlock()

	if ql.lineIndex >= 0 {
		t.lineMu.Lock()
		state := &t.lineStates[ql.lineIndex]
		if state.queries == nil {
			state.queries = graph.Queries{}
		}
		state.queries.Set(ql.Query, resultCount)
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
			n.queries.AddStatuses(ql.Query, statusCounts)
			n.mu.Unlock()
		}
	}

	t.applyRules(ctx, goalID, n, ql.depth+1)
}

func (n *nodeState) overLimit(limit int, class korrel8r.Class) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if limit > 0 && len(n.queries) > limit {
		log.V(5).Info("Query limit reached", "class", class, "queries", len(n.queries))
		return true
	}
	return false
}

// applyRules applies outgoing correlation rules to unprocessed objects in a node.
// The processed counter ensures each object is rule-applied exactly once,
// even when multiple goroutines call this concurrently for the same node.
func (t *traverser) applyRules(ctx context.Context, nodeID int64, n *nodeState, nextDepth int) {
	// Snapshot the objects, update processed, release the lock
	n.mu.Lock()
	objects := n.result.List()
	start := n.processed
	n.processed = len(objects)
	static := &t.nodeStatic[nodeID]
	class := static.class
	n.mu.Unlock()

	if start >= len(objects) {
		return
	}

	rules := static.rules
	for _, o := range objects[start:] {
		for _, r := range rules {
			if ctx.Err() != nil {
				return
			}
			queries, err := r.Apply(o)
			log.V(4).Info("Rule applied", "name", r.Name(), "start", class, "error", err, "queries", len(queries))
			metricRules.Add(ctx, 1, t.engine.RuleMetricAttrs(r))
			for _, q := range queries {
				goal := t.data.NodeFor(q.Class())
				if goal == nil {
					continue
				}
				key := lineKey{start: nodeID, rule: r, goal: goal.ID()}
				if index, ok := t.lineIndex[key]; ok {
					t.dedupAndSend(ctx, queryLine{Query: q, lineIndex: index, depth: nextDepth})
				}
			}
		}
	}
}
