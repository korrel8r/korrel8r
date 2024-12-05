// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph/topo"
)

// async processes stores concurrently.
type asyncTraverser struct {
	traverser
	buffer int
}

func NewAsync(e *engine.Engine, g *graph.Graph, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query) Traverser {
	// With unbuffered channels, the search may deadlock if the graph has a cycle.
	// Use buffers that are as long as the largest strongly connected component,
	// and therefore longer than the longest cycle.
	components := topo.TarjanSCC(g)
	longest := 0
	for _, c := range components {
		if len(c) > longest {
			longest = len(c)
		}
	}
	// If there are no cycles, all components will have size 1, and we can
	// use unbuffered channels.
	return &asyncTraverser{traverser: *newTraverser(e, g, start, objects, queries), buffer: longest - 1}
}

// Node implements [graph.Visitor.Node] by ensuring the node exists.
func (t *asyncTraverser) Node(n *graph.Node) { t.ensureNode(n) }

// Line implements [graph.Visitor.Line] to set up the sending count on the goal node.
func (t *asyncTraverser) Line(l *graph.Line) bool {
	t.ensureNode(l.Goal()).Sending() // Count each start->goal line as a potential sender.
	return true
}

// Edge implements [graph.Visitor.Edge] as no-op
func (t *asyncTraverser) Edge(*graph.Edge) {}

func (t *asyncTraverser) ensureNode(n *graph.Node) *node {
	if n.Value == nil {
		n.Value = &node{Node: n, asyncTraverser: t, queryChan: make(chan lineQuery, t.buffer)}
	}
	return getNode(n)
}

// Goals runs a goal-directed search, results are filled in on [graph.Node.Result], and [graph.Node.Queries], and [graph.Line.Queries].
func (t *asyncTraverser) Goals(ctx context.Context, goals []korrel8r.Class) (*graph.Graph, error) {
	log.V(4).Info("Async goal search", "start", t.start, "goals", goals)
	var err error
	t.graph, err = t.graph.Traverse(t.start, goals, t)
	if err != nil {
		return nil, err
	}
	return t.run(ctx)
}

// Goals runs a neighbourhood, results are filled in on [graph.Node.Result], and [graph.Node.Queries], and [graph.Line.Queries].
func (t *asyncTraverser) Neighbours(ctx context.Context, n int) (*graph.Graph, error) {
	log.V(4).Info("Async neighbours search", "start", t.start, "depth", n)
	var err error
	t.graph, err = t.graph.Neighbours(t.start, n, t)
	if err != nil {
		return nil, err
	}
	return t.run(ctx)
}

// run the concurrent graph traversal, return when it is complete.
func (t *asyncTraverser) run(ctx context.Context) (*graph.Graph, error) {
	// Add start objects to the start node, before activating.
	start := getNode(t.graph.NodeFor(t.start))
	start.Result.Append(t.objects...)
	start.Sending()          // Notify the start node that we are sending.
	busy := sync.WaitGroup{} // Wait for nodes

	defer func() { // Before returning from this function...
		start.Done() // Notify the start node we are done.
		busy.Wait()  // Wait for all nodes to finish.
	}()

	t.graph.EachNode(func(n *graph.Node) {
		busy.Add(1)
		go func() {
			defer busy.Done()
			getNode(n).Run(ctx)
		}()
	})

	// Send start queries to the start node
	for _, q := range t.queries {
		start.queryChan <- lineQuery{Query: q}
	}

	return t.graph, nil
}

// node holds the async traversal state for a single graph node.
type node struct {
	*graph.Node
	*asyncTraverser

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
func (n *node) Sending() {
	senders := n.senders.Add(1)
	log.V(4).Info("Sender added", "class", n.Class, "n", senders)
}

// Done informs the node that one of its senders is finished.
func (n *node) Done() int64 {
	senders := n.senders.Add(-1)
	log.V(4).Info("Sender removed", "class", n.Class, "n", senders)
	if senders == 0 {
		close(n.queryChan) // Run() will exit when channel is cleared.
	} else if senders < 0 {
		panic("Done called too many times.")
	}
	return senders
}

// Run handles incoming queries for a single node.
//
// It updates it's own graph node and lines incoming to that node with query and count data.
// No need to sync updates since each Run goroutine operates on a distinct data set.
func (n *node) Run(ctx context.Context) {
	defer func() { // On exit inform each goal nodes that we are done.
		n.graph.EachLineFrom(n.Node, func(l *graph.Line) { getNode(l.Goal()).Done() })
	}()

	// Handle start-up objects first if there are any.
	for _, o := range n.Result.List() {
		n.runObject(ctx, o)
	}

	// Now process the query channel.
	for lq := range n.queryChan {
		l, q := lq.Line, lq.Query
		if n.Queries.Has(q) {
			continue // Already processed this query.
		}
		count := 0
		process := korrel8r.AppenderFunc(func(o korrel8r.Object) {
			if n.Result.Add(o) { // Only process if this is a new object.
				count++
				n.runObject(ctx, o)
			}
		})
		_ = n.engine.Get(ctx, q, korrel8r.ConstraintFrom(ctx), process)
		n.Queries.Set(q, count)
		if l != nil { // Initial queries don't have a line
			l.Queries.Set(q, count)
		}
	}
}

func (n *node) runObject(ctx context.Context, o korrel8r.Object) {
	// Rules with multiple goal classes can appear on multiple outbound lines.
	// We don't know the actual goal class till the rule is first applied.
	// The rule may be first applied on the wrong line for the actual goal class.
	//
	// SO: remember rules applied on the wrong line, send them on the correct line.
	applied := map[korrel8r.Rule]struct {
		q   korrel8r.Query
		err error
	}{}
	n.graph.EachLineFrom(n.Node, func(l *graph.Line) {
		qe, ok := applied[l.Rule] // Already applied?
		if !ok {                  // No, apply now
			qe.q, qe.err = l.Rule.Apply(o)
		}
		if qe.q != nil && qe.q.Class() != l.Goal().Class { // Wrong line, save for later
			applied[l.Rule] = qe
			return
		}
		if qe.err != nil || qe.q == nil {
			log.V(4).Info("Cannot apply", "rule", l.Rule.Name(), "error", qe.err)
		} else {
			log.V(4).Info("Applied", "rule", l.Rule.Name(), "query", qe.q)
			getNode(l.Goal()).queryChan <- lineQuery{Query: qe.q, Line: l}
		}
	})
}

// FIXME cancellation overall - traversal/traverser visitor, follower? Error handling.
