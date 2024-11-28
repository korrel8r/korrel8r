// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"golang.org/x/exp/maps"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/traverse"
)

// Visitor used to traverse a graph.
type Visitor interface {
	// Node visits a node. Called after Edge/Line leading into the node, before Edge/Line leading out.
	Node(*Node)

	// Line a line. Return false if this line should not be traversed.
	Line(*Line) bool

	// Traverse an edge between two nodes.
	// Called after calling [Line] for each line in the edge, if at least one Line returned true.
	Edge(*Edge)
}

// LineVisitor adapts a line traversal function to implement the Visitor interface.
type LineVisitor func(*Line) bool

func (tf LineVisitor) Line(l *Line) bool { return tf(l) }
func (tf LineVisitor) Node(*Node)        {}
func (tf LineVisitor) Edge(*Edge)        {}

// Traverse rules on paths from start to goal.
// Returns the subset of the graph that was traversed.
func (g *Graph) Traverse(start korrel8r.Class, goals []korrel8r.Class, v Visitor) (*Graph, error) {
	sub := g.Data.EmptyGraph()
	bf := traverse.BreadthFirst{
		Traverse: func(e graph.Edge) bool {
			return g.traverseEdge(e, v.Edge, func(l *Line) bool {
				if v.Line(l) {
					sub.SetLine(l)
					return true
				}
				return false
			})
		},
		Visit: func(n graph.Node) { v.Node(NodeFor(n)) },
	}
	bf.Walk(g, g.NodeFor(start), nil)
	return sub, nil
}

// Neighbours traverses a breadth-first neighbourhood of start.
// Returns the subset of the graph that was traversed.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, v Visitor) (*Graph, error) {
	sub := g.Data.EmptyGraph()
	sub.AddNode(g.NodeFor(start))
	atDepth := 0
	current := unique.Set[int64]{} // Nodes at the current depth or above.

	bf := traverse.BreadthFirst{
		Traverse: func(edge graph.Edge) bool {
			return g.traverseEdge(edge, v.Edge, func(l *Line) bool {
				to := l.To().ID()
				// Process if we are not at depth, and the to node is in the current set or undiscovered.
				if atDepth < depth && (current.Has(to) || sub.Node(to) == nil) && v.Line(l) {
					sub.SetLine(l)
					return true
				}
				return false
			})
		},
		Visit: func(n graph.Node) {
			current.Add(n.ID())
			v.Node(NodeFor(n))
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
func (g *Graph) traverseEdge(ge graph.Edge, edgeFunc func(*Edge), lineFunc func(*Line) bool) (ok bool) {
	lines := g.Lines(ge.From().ID(), ge.To().ID())
	for lines.Next() {
		ok = lineFunc(lines.Line().(*Line)) || ok
	}
	if ok {
		e := Edge(ge.(multi.Edge))
		edgeFunc(&e)
	}
	return ok
}
