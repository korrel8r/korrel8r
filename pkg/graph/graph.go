// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package graph provides a directed multi-graph with class nodes and rule lines.
//
// Functions in this package manipulate rule graphs, e.g. finding paths or minimizing the graphs.
// They do not interrogate stores to find live correlations, for that see the [engine.Engine]
package graph

import (
	"fmt"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
)

// Graph is a directed multigraph with [korrel8r.Class] noes and [korrel8r.Rule] lines.
// Nodes and lines carry attributes for rendering by GraphViz.
//
// Concurrency: Graph is mutable, normal concurrency rules apply regarding read/write operations.
// The underlying [Data] is immutable, but the lines and nodes included in the Graph can change.
type Graph struct {
	*multi.DirectedGraph
	GraphAttrs, NodeAttrs, EdgeAttrs Attrs
	Data                             *Data
}

// New empty graph based on Data
func New(data *Data) *Graph {
	return &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
		GraphAttrs: Attrs{
			"fontname":        "Helvetica",
			"fontsize":        "12",
			"splines":         "true",
			"overlap":         "prism",
			"overlap_scaling": "-2",
			"layout":          "dot",
		},
		NodeAttrs: Attrs{
			"fontname": "Helvetica",
			"fontsize": "12",
		},
		EdgeAttrs: Attrs{
			"fontname": "Helvetica",
			"fontsize": "12",
		},
		Data: data,
	}
}

func (g *Graph) NodeFor(c korrel8r.Class) *Node {
	n := g.Data.NodeFor(c)
	if n == nil || g.Node(n.ID()) == nil {
		return nil
	}
	return n
}

func (g *Graph) EachNode(visit func(*Node)) {
	nodes := g.Nodes()
	for nodes.Next() {
		visit(nodes.Node().(*Node))
	}
}

func (g *Graph) EachEdge(visit func(*Edge)) {
	edges := g.Edges()
	for edges.Next() {
		edge := Edge{Edge: edges.Edge().(multi.Edge)}
		visit(&edge)
	}
}

func (g *Graph) EachLine(visit func(*Line)) {
	edges := g.Edges()
	for edges.Next() {
		e := edges.Edge()
		g.EachLineBetween(e.From().(*Node), e.To().(*Node), visit)
	}
}

// EachLineBetween calls visit(l) for each line between start and goal (if there are any).
func (g *Graph) EachLineBetween(start, goal *Node, visit func(l *Line)) {
	// NOTE: do not use embedded [multi.Edge.Lines] iterator, it modifies the edge, concurrent unsafe.
	// Instead create a new iterator with [graph.Lines]
	lines := g.Lines(start.ID(), goal.ID())
	for lines.Next() {
		visit(lines.Line().(*Line))
	}
}

// EachLineFrom calls visit(l) for each line from start.
func (g *Graph) EachLineFrom(start *Node, visit func(*Line)) {
	goals := g.From(start.ID())
	for goals.Next() {
		g.EachLineBetween(start, goals.Node().(*Node), visit)
	}
}

// EachLineTo calls visit(l) for each line to goal.
func (g *Graph) EachLineTo(goal *Node, visit func(*Line)) {
	starts := g.To(goal.ID())
	for starts.Next() {
		g.EachLineBetween(starts.Node().(*Node), goal, visit)
	}
}

// Select creates a sub-graph of all lines where keep(line) is true.
func (g *Graph) Select(keep func(*Line) bool) *Graph {
	sub := g.Data.EmptyGraph()
	g.EachLine(func(l *Line) {
		if keep(l) {
			sub.SetLine(l)
		}
	})
	return sub
}

func (g *Graph) DOTID() string { return g.GraphAttrs["name"] }
func (g *Graph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	return g.GraphAttrs, g.NodeAttrs, g.EdgeAttrs
}

// ShortestPaths returns a new sub-graph containing all shortest paths between start and goals.
func (g *Graph) ShortestPaths(start korrel8r.Class, goals ...korrel8r.Class) *Graph {
	paths := path.DijkstraAllFrom(g.NodeFor(start), g)
	sub := g.Data.EmptyGraph()
	for _, goal := range goals {
		n := g.NodeFor(goal)
		if n == nil {
			log.V(1).Info("Goal not in graph", "class", goal)
			continue
		}
		v := n.ID()
		paths.AllToFunc(v, func(path []graph.Node) {
			for i := 1; i < len(path); i++ {
				lines := g.Lines(path[i-1].ID(), path[i].ID())
				for lines.Next() {
					sub.SetLine(lines.Line())
				}
			}
		})
	}
	return sub
}

// GoalSearch traverses the shortest paths from start to all goals, in breadth first order.
func (g *Graph) GoalSearch(start korrel8r.Class, goals []korrel8r.Class, v Visitor) {
	g.ShortestPaths(start, goals...).BreadthFirst(start, v, nil)
}

// Neighbours traverses a breadth-first neighbourhood of start up to depth.
func (g *Graph) Neighbours(startClass korrel8r.Class, maxDepth int, v Visitor) {
	start := g.NodeFor(startClass)
	if start == nil {
		return
	}
	depth := 0
	line := func(l *Line) bool {
		if depth < maxDepth {
			return v.Line(l)
		}
		return false
	}
	until := func(n *Node, d int) bool { depth = d; return d > maxDepth }
	g.BreadthFirst(startClass, FuncVisitor{LineF: line, NodeF: v.Node}, until)
}

// FindLine finds a line between start and goal with rule. Nil if not found.
func (g *Graph) FindLine(start, goal korrel8r.Class, rule korrel8r.Rule) *Line {
	u, v := g.NodeFor(start), g.NodeFor(goal)
	if u == nil || v == nil {
		return nil
	}
	lines := g.Lines(u.ID(), v.ID())
	for lines.Next() {
		l := lines.Line().(*Line)
		if l.Rule == rule {
			return l
		}
	}
	return nil
}

func (g *Graph) String() string {
	w := &strings.Builder{}
	fmt.Fprintln(w, "<Graph>")
	g.EachLine(func(l *Line) {
		fmt.Fprintln(w, l.String())
	})
	fmt.Fprintln(w, "</Graph>")
	return w.String()
}

// Pull useful functions into this package.
var (
	NodesOf = graph.NodesOf
	LinesOf = graph.LinesOf
	EdgesOf = graph.EdgesOf
)

var log = logging.Log()
