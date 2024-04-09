// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

// Visitor interface to visit nodes and traverse lines during a graph traversal.
type Visitor interface {
	// Visit is called for each node, with an iterator for the lines from this node.
	Visit(n *Node, from Lines)
	Traverse(*Line)
}

// Lines iterates over a set of lnes, calling the traverse function for each one.
type Lines func(traverse func(*Line))

// MakeVisitor makes a visitor from a pair of functions.
// A nil argument will be converted to a no-op function.
func MakeVisitor(visit func(*Node, Lines), traverse func(*Line)) Visitor {
	if visit == nil {
		visit = func(*Node, Lines) {}
	}
	if traverse == nil {
		traverse = func(*Line) {}
	}
	return visitorFuncs{visit: visit, traverse: traverse}
}

type visitorFuncs struct {
	visit    func(*Node, Lines)
	traverse func(*Line)
}

func (vf visitorFuncs) Visit(n *Node, from Lines) { vf.visit(n, from) }
func (vf visitorFuncs) Traverse(l *Line)          { vf.traverse(l) }

// Traverse traverses rules on a path graph in topological order.
//
// Cyclic components are traversed in topological order, but rules within the cycle are traversed in arbitrary order.
func (g *Graph) Traverse(v Visitor) error {
	ordered, err := topo.Sort(g)
	cycles, ok := err.(topo.Unorderable)
	if err != nil && !ok {
		return err
	}
	j := 0 // cycles index
	for _, n := range ordered {
		if n != nil {
			from := func(f func(l *Line)) { g.traverseFrom(n, f) }
			v.Visit(n.(*Node), from)
			from(v.Traverse)
		} else {
			inside := make(unique.Set[int64], len(cycles[j]))
			for _, n := range cycles[j] {
				inside.Add(n.ID())
			}
			for _, n := range cycles[j] {
				from := func(f func(l *Line)) { g.traverseFrom(n, f) }
				v.Visit(n.(*Node), from)
				// Visit lines inside the cycle before those that leave it.
				from(func(l *Line) {
					if inside.Has(l.To().ID()) {
						v.Traverse(l)
					}
				})
				from(func(l *Line) {
					if !inside.Has(l.To().ID()) {
						v.Traverse(l)
					}
				})
			}
			j++
		}
	}
	return nil
}

// Neighbours creates a neighbourhood graph around start and traverses rules breadth-first.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, v Visitor) *Graph {
	depths := map[int64]int{}
	var nodes []graph.Node
	sub := g.Data.EmptyGraph()
	bf := traverse.BreadthFirst{}
	bf.Walk(g, g.NodeFor(start), func(n graph.Node, d int) bool {
		if d > depth {
			return true
		}
		nodes = append(nodes, n)
		sub.AddNode(n)
		if d2, ok := depths[n.ID()]; !ok || d2 > d { // Record shortest path depth to n
			depths[n.ID()] = d
		}
		return false
	})
	for _, n := range nodes {
		from := func(f func(l *Line)) { g.traverseFrom(n, f) }
		v.Visit(n.(*Node), from)
		g.traverseTo(n, func(l *Line) { // Add lines from lesser to greater depth
			if depths[l.From().ID()] < depths[l.To().ID()] {
				sub.SetLine(l)
				v.Traverse(l)
			}
		})
	}
	return sub
}

// traverseFrom traverses each edge from node.
func (g *Graph) traverseFrom(node graph.Node, traverse func(*Line)) {
	to := g.From(node.ID())
	for to.Next() {
		g.traverseLines(g.Lines(node.ID(), to.Node().ID()), traverse)
	}
}

// traverseTo traverses each edge to node.
func (g *Graph) traverseTo(node graph.Node, traverse func(l *Line)) {
	from := g.To(node.ID())
	for from.Next() {
		g.traverseLines(g.Lines(from.Node().ID(), node.ID()), traverse)
	}
}

// traverseLines calls travers on each line.
func (g *Graph) traverseLines(lines graph.Lines, traverse func(l *Line)) {
	for lines.Next() {
		traverse(lines.Line().(*Line))
	}
}
