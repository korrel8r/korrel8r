// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

// Traverse traverses rules on a path graph in topological order.
//
// Cyclic components are traversed in topological order, but rules within the cycle are traversed in arbitrary order.
func (g *Graph) Traverse(traverse func(l *Line)) error {
	ordered, err := topo.Sort(g)
	cycles, ok := err.(topo.Unorderable)
	if err != nil && !ok {
		return err
	}
	j := 0 // cycles index
	for _, n := range ordered {
		if n != nil {
			g.traverseFrom(n, traverse)
		} else {
			inside := make(unique.Set[int64], len(cycles[j]))
			for _, n := range cycles[j] {
				inside.Add(n.ID())
			}
			for _, n := range cycles[j] {
				// Visit lines inside the cycle before those that leave it.
				g.traverseFrom(n, func(l *Line) {
					if inside.Has(l.To().ID()) {
						traverse(l)
					}
				})
				g.traverseFrom(n, func(l *Line) {
					if !inside.Has(l.To().ID()) {
						traverse(l)
					}
				})
			}
			j++
		}
	}
	return nil
}

// Neighbours creates a neighbourhood graph around start and traverses rules breadth-first.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, travers func(l *Line)) *Graph {
	depths := map[int64]int{}
	var nodes []graph.Node
	sub := g.Data.EmptyGraph()
	bf := traverse.BreadthFirst{Visit: func(n graph.Node) {
		nodes = append(nodes, n)
		sub.AddNode(n)
	}}
	bf.Walk(g, g.NodeFor(start), func(n graph.Node, d int) bool {
		if d > depth {
			return true
		}
		if d2, ok := depths[n.ID()]; !ok || d2 > d { // Record shortest path depth to n
			depths[n.ID()] = d
		}
		return false
	})
	for _, n := range nodes {
		g.traverseTo(n, func(l *Line) { // Add lines from lesser to greater depth
			if depths[l.From().ID()] < depths[l.To().ID()] {
				sub.SetLine(l)
				travers(l)
			}
		})
	}
	return sub
}

// traverseFrom traverses each edge from node.
func (g *Graph) traverseFrom(node graph.Node, traverse func(l *Line)) {
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
