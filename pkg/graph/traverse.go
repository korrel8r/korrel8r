// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/traverse"
)

type edgeFollower struct {
	Graph, SubGraph *Graph
	F               func(l *Line) bool
}

func (ef *edgeFollower) Traverse(edge graph.Edge) bool {
	return ef.Graph.traverseLines(ef.Graph.Lines(edge.From().ID(), edge.To().ID()), func(l *Line) bool {
		if ef.F(l) {
			ef.SubGraph.SetLine(l)
			return true
		}
		return false
	})
}

// Traverse rules on paths from start to goal.
//
// Cyclic components are traversed in topological order, but rules within the cycle are traversed in arbitrary order.
func (g *Graph) Traverse(start korrel8r.Class, goals []korrel8r.Class, f func(*Line) bool) *Graph {
	ef := edgeFollower{Graph: g, SubGraph: g.Data.EmptyGraph(), F: f}
	bf := traverse.BreadthFirst{Traverse: ef.Traverse}
	bf.Walk(g, g.NodeFor(start), nil)
	return ef.SubGraph
}

// Neighbours creates a neighbourhood graph around start and traverses rules breadth-first.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, f func(*Line) bool) *Graph {
	ef := edgeFollower{Graph: g, SubGraph: g.Data.EmptyGraph(), F: f}
	bf := traverse.BreadthFirst{Traverse: ef.Traverse}
	bf.Walk(g, g.NodeFor(start), func(n graph.Node, d int) bool { return d >= depth })
	return ef.SubGraph
}

// traverseLines calls travers on each line.
func (g *Graph) traverseLines(lines graph.Lines, f func(l *Line) bool) (ok bool) {
	for lines.Next() {
		ok = f(lines.Line().(*Line)) || ok
	}
	return ok
}
