// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"math"
	"slices"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/simple"
)

// View adapts Data's compact topology to Gonum's read-only weighted directed
// graph interface. It owns no Node/Line objects or cached graph representation.
// Nodes are lightweight IDs, not result-bearing *Node objects. Parallel lines
// form one edge whose weight is the least rule spread, as in Graph.Weight.
// Views and Data can be read concurrently; each iterator has independent state.
type View struct{ data *Data }

var _ graph.WeightedDirected = View{}

// Graph returns a read-only Gonum view without building or caching another graph.
func (d *Data) Graph() View { return View{data: d} }

func (g View) Node(id int64) graph.Node {
	if id < 0 || id >= int64(g.data.NodeCount()) {
		return nil
	}
	return multi.Node(id)
}

func (g View) Nodes() graph.Nodes        { return &viewNodes{count: g.data.NodeCount(), current: -1} }
func (g View) From(id int64) graph.Nodes { return g.neighbors(id, false) }
func (g View) To(id int64) graph.Nodes   { return g.neighbors(id, true) }

// Deduplicate on demand rather than retaining a second adjacency representation.
// Sorting gives stable neighbor order even with parallel lines and repeated rules.
func (g View) neighbors(id int64, incoming bool) graph.Nodes {
	if g.Node(id) == nil {
		return graph.Empty
	}
	a := g.data.topology.out
	if incoming {
		a = g.data.topology.in
	}
	lines := a.lines[a.offsets[id]:a.offsets[id+1]]
	if len(lines) == 0 {
		return graph.Empty
	}
	ids := make([]int, len(lines))
	for i, lineID := range lines {
		l := g.data.topology.lines[lineID]
		ids[i] = int(l.to)
		if incoming {
			ids[i] = int(l.from)
		}
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	return &viewNodes{ids: ids, count: len(ids), current: -1}
}

// Edge lookups use Data's pre-computed edge index, so they do not scan the
// source node's adjacency. Shortest-path searches relax every edge repeatedly,
// so a scan here would cost O(out-degree) per relaxation.
func (g View) HasEdgeFromTo(from, to int64) bool {
	_, ok := g.data.EdgeWeight(from, to)
	return ok
}
func (g View) HasEdgeBetween(x, y int64) bool { return g.HasEdgeFromTo(x, y) || g.HasEdgeFromTo(y, x) }
func (g View) Edge(from, to int64) graph.Edge { return g.WeightedEdge(from, to) }
func (g View) WeightedEdge(from, to int64) graph.WeightedEdge {
	if !g.HasEdgeFromTo(from, to) {
		return nil
	}
	w, _ := g.Weight(from, to) // Not EdgeWeight: self-loops keep Graph.Weight's zero weight.
	return simple.WeightedEdge{F: multi.Node(from), T: multi.Node(to), W: w}
}

// Weight preserves Graph.Weight's zero self-weight (even for an absent node).
// A self-weight does not imply that a self-loop edge exists.
func (g View) Weight(from, to int64) (float64, bool) {
	if from == to {
		return 0, true
	}
	if w, ok := g.data.EdgeWeight(from, to); ok {
		return w, true
	}
	return math.Inf(1), false
}

// viewNodes iterates either explicit neighbor IDs or the dense node-ID range.
// It does not expose its backing slice, and Reset affects only this iterator.
type viewNodes struct {
	ids                  []int
	count, next, current int
}

func (it *viewNodes) Next() bool {
	if it.next == it.count {
		it.current = -1
		return false
	}
	it.current = it.next
	it.next++
	return true
}
func (it *viewNodes) Len() int { return it.count - it.next }
func (it *viewNodes) Reset()   { it.next, it.current = 0, -1 }
func (it *viewNodes) Node() graph.Node {
	if it.current < 0 {
		return nil
	}
	id := it.current
	if it.ids != nil {
		id = it.ids[id]
	}
	return multi.Node(id)
}
