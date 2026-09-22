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
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"go.opentelemetry.io/otel/metric"
	"gonum.org/v1/gonum/graph/path"
)

// Goals traverses all paths from start objects to all goal classes.
func Goals(ctx context.Context, e *engine.Engine, start Start, goals []korrel8r.Class) (*graph.Graph, error) {
	log.V(2).Info("Goal directed search", "start", start, "goals", goals, "constraint", start.Constraint)
	data := e.GraphData()
	scope, err := goalScope(data, start.Class, goals)
	if err != nil {
		return nil, err
	}
	g, err := newTraverser(e, data, scope, start.Constraint, -1).run(ctx, start)
	// Remove dead-end paths that don't reach a goal, including in partial results.
	g.RemoveEmptyGoalPaths(goals)
	return g, err
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

// neighborScope returns line IDs reachable within maxDepth BFS hops from start.
func neighborScope(data *graph.Data, start korrel8r.Class, maxDepth int) ([]int, error) {
	startID, ok := data.NodeID(start)
	if !ok {
		return nil, fmt.Errorf("class not found in graph: %v", start)
	}

	if maxDepth <= 0 {
		return nil, nil
	}
	// Node IDs are dense. Zero means unseen; stored depths are offset by one.
	nodeDepth := make([]int, data.NodeCount())
	nodeDepth[startID] = 1
	queue := make([]int64, 1, data.NodeCount())
	queue[0] = startID
	candidates := 0
	for head := 0; head < len(queue); head++ {
		id := queue[head]
		depth := nodeDepth[id] - 1
		if depth >= maxDepth {
			continue
		}
		data.EachLineIDFrom(id, func(l int) {
			candidates++
			_, goalID := data.Endpoints(l)
			if nodeDepth[goalID] == 0 {
				nodeDepth[goalID] = nodeDepth[id] + 1
				queue = append(queue, goalID)
			}
		})
	}

	// Revisit adjacency in the same BFS order, now with all depths known.
	// The first pass sizes the buffer, avoiding repeated scope-slice growth.
	lines := make([]int, 0, candidates)
	for _, id := range queue {
		if nodeDepth[id]-1 >= maxDepth {
			continue
		}
		data.EachLineIDFrom(id, func(l int) {
			_, goalID := data.Endpoints(l)
			if nodeDepth[id] <= nodeDepth[goalID] {
				lines = append(lines, l)
			}
		})
	}
	return lines, nil
}

// goalScope returns the lines on shortest/near-shortest paths from start to each goal.
func goalScope(data *graph.Data, start korrel8r.Class, goals []korrel8r.Class) ([]int, error) {
	startID, ok := data.NodeID(start)
	if !ok {
		return nil, fmt.Errorf("class not found in graph: %v", start)
	}
	view := data.Graph()
	var lines []int
	for _, goal := range goals {
		goalID, ok := data.NodeID(goal)
		if !ok {
			return nil, fmt.Errorf("class not found in graph: %v", goal)
		}
		paths := path.YenKShortestPaths(view, math.MaxInt, 1, view.Node(startID), view.Node(goalID))
		for _, p := range paths {
			for i := 1; i < len(p); i++ {
				data.EachLineIDFrom(p[i-1].ID(), func(id int) {
					_, to := data.Endpoints(id)
					if to == p[i].ID() {
						lines = append(lines, id)
					}
				})
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
	Query  korrel8r.Query
	lineID int // topology line ID; -1 for a start query
	depth  int
}

// LimitError reports that a traversal returned a partial graph after exhausting a total budget.
type LimitError struct {
	Name  string
	Limit int
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("traversal %s exceeded: limit %d", e.Name, e.Limit)
}

// initLine returns the stable, search-owned line for a topology ID.
// The caller must hold lineMu while workers are active.
func (t *traverser) initLine(id int) *graph.Line {
	if t.lines == nil {
		t.lines = make(map[int]*graph.Line)
	}
	if t.lines[id] == nil {
		from, to := t.data.Endpoints(id)
		t.lines[id] = t.data.NewLine(id, t.getOrCreateNodeState(from).Node, t.getOrCreateNodeState(to).Node)
	}
	return t.lines[id]
}

// scopedLine resolves a generated query using scoped topology IDs. lines is sorted
// by goal, so only the lines sharing this goal are examined. Reverse iteration
// within the group preserves last-scoped-line-wins for duplicate endpoint/rule
// keys, including repeated IDs and scopes in a different order from topology creation.
func (t *traverser) scopedLine(start, goal int64, rule korrel8r.Rule) int {
	lines := t.nodeStatic[start].lines
	lo, _ := slices.BinarySearchFunc(lines, goal, func(id int, goal int64) int {
		_, to := t.data.Endpoints(id)
		return cmp.Compare(to, goal)
	})
	last := -1
	for i := lo; i < len(lines); i++ {
		id := lines[i]
		if _, to := t.data.Endpoints(id); to != goal {
			break
		}
		if t.data.RuleForLine(id) == rule {
			last = id
		}
	}
	return last
}

// nodeStatic holds class identity and immutable search routing metadata,
// indexed by node ID. It contains no mutable result node.
type nodeStatic struct {
	class       korrel8r.Class
	rules       []korrel8r.Rule
	lines       []int // Scoped outgoing IDs sorted by goal; shared contiguous backing.
	classMetric metric.MeasurementOption
}

// nodeState owns a search-owned node and its synchronization/processing state.
// It is allocated only when the class becomes active in this search.
type nodeState struct {
	mu          sync.Mutex
	*graph.Node     // Search-owned result node; reused by the returned Graph.
	processed   int // count of result objects already rule-applied
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
	nodeStatic      []nodeStatic // Immutable routing, indexed by graph node ID.
	totalLimit      int
	totalQueryLimit int

	// Concurrent state
	nodeMu       sync.Mutex
	nodeState    []*nodeState // Lazy mutable overlays, indexed by graph node ID.
	work         *workQueue
	wg           sync.WaitGroup
	budgetMu     sync.Mutex
	totalObjects int
	limitOnce    sync.Once
	limitErr     *LimitError
	stopped      atomic.Bool
	seenMu       sync.Mutex
	seen         map[korrel8r.Query]struct{}
	lineMu       sync.Mutex
	lines        map[int]*graph.Line // Only lines with completed queries; search-owned.
}

func newTraverser(e *engine.Engine, data *graph.Data, scopeLines []int, c *korrel8r.Constraint, maxDepth int) *traverser {
	t := &traverser{
		engine:          e,
		data:            data,
		constraint:      c,
		maxDepth:        maxDepth,
		nodeStatic:      make([]nodeStatic, data.NodeCount()),
		nodeState:       make([]*nodeState, data.NodeCount()),
		work:            newWorkQueue(),
		seen:            map[korrel8r.Query]struct{}{},
		totalLimit:      effectiveLimit(e.Tuning.TotalLimit, c.GetTotalLimit()),
		totalQueryLimit: effectiveLimit(e.Tuning.TotalQueryLimit, c.GetTotalQueryLimit()),
	}

	// Partition one ID buffer into per-node routing slices. Storage scales with
	// the scope, not the entire topology, so narrow goal searches stay small.
	counts := make([]int, data.NodeCount())
	for _, id := range scopeLines {
		start, _ := data.Endpoints(id)
		counts[start]++
	}
	ids := make([]int, len(scopeLines))
	offset := 0
	for node, count := range counts {
		t.nodeStatic[node].lines = ids[offset : offset : offset+count]
		offset += count
	}
	for _, id := range scopeLines {
		startID, goalID := data.Endpoints(id)
		rule := data.RuleForLine(id)
		t.initNodeStatic(startID, data.Class(startID))
		t.initNodeStatic(goalID, data.Class(goalID))
		start := &t.nodeStatic[startID]
		if !slices.Contains(start.rules, rule) {
			start.rules = append(start.rules, rule)
		}
		start.lines = append(start.lines, id)
	}
	// Sort each node's lines by goal so scopedLine can binary-search the goal group
	// instead of scanning the whole (potentially very wide) out-adjacency.
	// A stable sort keeps scope order within a group, for last-scoped-line-wins.
	for node := range t.nodeStatic {
		slices.SortStableFunc(t.nodeStatic[node].lines, func(a, b int) int {
			_, ga := data.Endpoints(a)
			_, gb := data.Endpoints(b)
			return cmp.Compare(ga, gb)
		})
	}

	return t
}

func effectiveLimit(server, request int) int {
	if server <= 0 {
		return request
	}
	if request <= 0 {
		return server
	}
	return min(server, request)
}

func (t *traverser) exceed(ctx context.Context, name string, limit int, attrs metric.MeasurementOption) {
	t.limitOnce.Do(func() {
		metricLimitExceeded.Add(ctx, 1, attrs)
		t.limitErr = &LimitError{Name: name, Limit: limit}
		t.stopped.Store(true)
	})
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
		n = &nodeState{Node: t.data.NewNode(id)}
		t.nodeState[id] = n
	}
	return n
}

// run launches the worker pool, primes start data, and waits for completion.
func (t *traverser) run(ctx context.Context, start Start) (*graph.Graph, error) {
	startID, _ := t.data.NodeID(start.Class) // Scope construction validated the class.
	t.initNodeStatic(startID, start.Class)
	startNode := t.getOrCreateNodeState(startID)

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
	before := len(startNode.Result.List())
	for _, object := range start.Objects {
		if !t.addObject(ctx, startNode, object) {
			break
		}
	}
	added := len(startNode.Result.List()) - before
	startNode.mu.Unlock()
	metricRetainedObjects.Add(ctx, int64(added))

	for _, q := range start.Queries {
		t.dedupAndSend(ctx, queryLine{Query: q, lineID: -1, depth: 0})
	}

	t.applyRules(ctx, startID, startNode, 1)

	t.wg.Done() // Release sentinel.
	t.wg.Wait()
	t.work.close()
	workerWg.Wait()

	g := t.buildGraph()
	cause := context.Cause(ctx)
	if cause == nil && t.limitErr != nil {
		cause = t.limitErr
	}
	if limitErr, ok := errors.AsType[*LimitError](cause); ok {
		g.Truncation = &graph.Truncation{Condition: limitErr.Name, Limit: limitErr.Limit}
	}
	return g, cause
}

// buildGraph creates a result graph containing only nodes and lines that produced results.
func (t *traverser) buildGraph() *graph.Graph {
	g := graph.New(t.data)
	for _, n := range t.nodeState {
		if n != nil && !n.Empty() {
			g.AddNode(n.Node)
		}
	}
	// Stable result insertion order, independent of concurrent completion order.
	lineIDs := make([]int, 0, len(t.lines))
	for id := range t.lines {
		lineIDs = append(lineIDs, id)
	}
	slices.Sort(lineIDs)
	for _, id := range lineIDs {
		l := t.lines[id]
		if l == nil || l.Queries.Total() == 0 {
			continue
		}
		if g.Node(l.From().ID()) == nil || g.Node(l.To().ID()) == nil {
			continue
		}
		// Workers have finished. Nodes and lines already belong to this search;
		// transfer them without copying state or rebinding endpoints.
		g.AddLine(l)
	}
	return g
}

// dedupAndSend checks depth and query dedup, then adds to the work queue.
func (t *traverser) dedupAndSend(ctx context.Context, ql queryLine) {
	if t.maxDepth >= 0 && ql.depth > t.maxDepth {
		return
	}
	if ctx.Err() != nil || t.stopped.Load() {
		return
	}
	if t.isDuplicate(ctx, ql) {
		return
	}
	t.wg.Add(1)
	t.work.put(ql)
}

// addObject adds a unique object to n's result, charged against the total object
// budget. The caller must hold n.mu, and must report the number of objects added
// to metricRetainedObjects. With no total budget there is nothing shared to
// protect, so workers are not serialized on budgetMu for every object.
func (t *traverser) addObject(ctx context.Context, n *nodeState, object korrel8r.Object) bool {
	if t.totalLimit <= 0 {
		n.Result.Add(object)
		return true
	}
	t.budgetMu.Lock()
	defer t.budgetMu.Unlock()
	if n.Result.Contains(object) {
		return true
	}
	if t.totalObjects >= t.totalLimit {
		t.exceed(ctx, "totalLimit", t.totalLimit, metricTotalLimit)
		return false
	}
	if n.Result.Add(object) {
		t.totalObjects++
	}
	return true
}

func (t *traverser) isDuplicate(ctx context.Context, ql queryLine) bool {
	t.seenMu.Lock()
	defer t.seenMu.Unlock()
	if _, exists := t.seen[ql.Query]; exists {
		id, _ := t.data.NodeID(ql.Query.Class())
		metricDuplicateQueries.Add(ctx, 1, t.nodeStatic[id].classMetric)
		return true
	}
	if t.totalQueryLimit > 0 && len(t.seen) >= t.totalQueryLimit {
		t.exceed(ctx, "totalQueryLimit", t.totalQueryLimit, metricTotalQueryLimit)
		return true
	}
	t.seen[ql.Query] = struct{}{}
	metricAcceptedQueries.Add(ctx, 1)
	return false
}

// handleQuery processes a single queryLine: executes the query, adds results, applies rules.
func (t *traverser) handleQuery(ctx context.Context, ql *queryLine) {
	defer t.wg.Done()
	if ctx.Err() != nil {
		return
	}

	goalClass := ql.Query.Class()
	goalID, ok := t.data.NodeID(goalClass)
	if !ok { // Class is not in the rule graph, drop the query.
		return
	}
	n := t.getOrCreateNodeState(goalID)
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
	before := len(n.Result.List())
	for _, o := range results {
		if !t.addObject(ctx, n, o) {
			break
		}
	}
	resultList := n.Result.List()
	resultCount := len(resultList) - before
	n.Queries.Set(ql.Query, resultCount)
	n.mu.Unlock()
	metricRetainedObjects.Add(ctx, int64(resultCount))

	if ql.lineID >= 0 {
		t.lineMu.Lock()
		t.initLine(ql.lineID).Queries.Set(ql.Query, resultCount)
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

	t.applyRules(ctx, goalID, n, ql.depth+1)
}

func (n *nodeState) overLimit(limit int, class korrel8r.Class) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if limit > 0 && len(n.Queries) > limit {
		log.V(5).Info("Query limit reached", "class", class, "queries", len(n.Queries))
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
	objects := n.Result.List()
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
				goalID, ok := t.data.NodeID(q.Class())
				if !ok {
					continue
				}
				if id := t.scopedLine(nodeID, goalID, r); id >= 0 {
					t.dedupAndSend(ctx, queryLine{Query: q, lineID: id, depth: nextDepth})
				}
			}
		}
	}
}
