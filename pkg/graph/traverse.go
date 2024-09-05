// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"golang.org/x/exp/maps"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/traverse"
)

// Traverse rules on paths from start to goal.
// Returns the subset of the graph that was traversed.
func (g *Graph) Traverse(start korrel8r.Class, goals []korrel8r.Class, f func(*Line) bool) (*Graph, error) {
	sub := g.Data.EmptyGraph()
	bf := traverse.BreadthFirst{
		Traverse: func(edge graph.Edge) bool {
			return g.traverseEdge(edge, func(l *Line) bool {
				if f(l) {
					sub.SetLine(l)
					return true
				}
				return false
			})
		}}
	bf.Walk(g, g.NodeFor(start), nil)
	return sub, nil
}

// Neighbours traverses a breadth-first neighbourhood of start.
// Returns the subset of the graph that was traversed.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, f func(*Line) bool) (*Graph, error) {
	sub := g.Data.EmptyGraph()
	sub.AddNode(g.NodeFor(start))
	atDepth := 0
	current := unique.Set[int64]{} // Nodes at the current depth or above.

	bf := traverse.BreadthFirst{
		Traverse: func(edge graph.Edge) bool {
			return g.traverseEdge(edge, func(l *Line) bool {
				to := l.To().ID()
				// Process if we are not at depth, and the to node is in the current set or undiscovered.
				if atDepth < depth && (current.Has(to) || sub.Node(to) == nil) && f(l) {
					sub.SetLine(l)
					return true
				}
				return false
			})
		},
		Visit: func(n graph.Node) {
			current.Add(n.ID())
		},
	}
	bf.Walk(g, g.NodeFor(start), func(n graph.Node, d int) bool {
		if d > atDepth {
			maps.Clear(current)
			atDepth = d
		}
		return d > depth
	})
	return sub, nil
}

// traverseEdge calls f(l) for each line l in the edge.
// Returns true if any call to f(l) returns true.
func (g *Graph) traverseEdge(edge graph.Edge, f func(l *Line) bool) (ok bool) {
	lines := g.Lines(edge.From().ID(), edge.To().ID())
	for lines.Next() {
		ok = f(lines.Line().(*Line)) || ok
	}
	return ok
}
