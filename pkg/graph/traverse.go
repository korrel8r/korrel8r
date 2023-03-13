package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
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
// If traverse == nil, just create the neighbourhood graph.
func (g *Graph) Neighbours(start korrel8r.Class, depth int, travers func(l *Line)) *Graph {
	sub := g.Data.EmptyGraph()
	if travers == nil {
		travers = func(l *Line) {}
	}
	bf := traverse.BreadthFirst{
		Traverse: func(e graph.Edge) bool {
			g.traverseLines(e.(multi.Edge), func(l *Line) { sub.SetLine(l); travers(l) })
			return true
		},
	}
	bf.Walk(g, g.NodeFor(start), func(n graph.Node, d int) bool { return d == depth })
	return sub
}

// traverseFrom traverses each edge from node, returns true if any edge returns true.
func (g *Graph) traverseFrom(node graph.Node, traverse func(l *Line)) {
	goals := g.From(node.ID())
	for goals.Next() {
		g.traverseLines(g.Edge(node.ID(), goals.Node().ID()).(multi.Edge), traverse)
	}
}

// traverseLines traverses each line, returns true if any line returns true.
func (g *Graph) traverseLines(lines graph.Lines, traverse func(l *Line)) {
	for lines.Next() {
		traverse(lines.Line().(*Line))
	}
}
