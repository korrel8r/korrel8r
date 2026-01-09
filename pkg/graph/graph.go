// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package graph provides a directed multi-graph with class nodes and rule lines.
//
// Functions in this package manipulate rule graphs, e.g. finding paths or minimizing the graphs.
// They do not interrogate stores to find live correlations, for that see the [engine.Engine]
package graph

import (
	"fmt"
	"math"
	"slices"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/traverse"
)

// Graph is a directed multigraph with [korrel8r.Class] nodes and [korrel8r.Rule] lines.
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

// Weight an edge by the "spread" of its rules.
//
// Wildcard rules in domains with many classes (e.g. k8s, DependentToOwner) are "expensive".
// They create many speculative graph lines, but following these lines often leads nowhere.
// Weight edges based on the least expensive rule, in other words avoid
// consider an edge expensive if it has with only expensive rules.
func (g *Graph) Weight(u, v int64) (w float64, ok bool) {
	if u == v {
		return 0, true
	}
	l := g.Lines(u, v)
	w = math.Inf(1)
	for l.Next() {
		ok = true
		goals := l.Line().(*Line).Rule.Goal()
		w = math.Min(w, float64(len(goals)))
	}
	return w, ok
}

func (g *Graph) NodeFor(c korrel8r.Class) *Node {
	n := g.Data.NodeFor(c)
	if n == nil || g.Node(n.ID()) == nil {
		return nil
	}
	return n
}

func (g *Graph) NodeForErr(c korrel8r.Class) (*Node, error) {
	n := g.NodeFor(c)
	if n == nil {
		return nil, fmt.Errorf("class not found in graph: %v", c)
	}
	return n, nil
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

// GoalPaths returns a new sub-graph containing only nodes on a path to the goal class.
// Includes k-shortest paths with cost <= shortest+1.
func (g *Graph) GoalPaths(start korrel8r.Class, goals []korrel8r.Class) (*Graph, error) {
	u, err := g.NodeForErr(start)
	if err != nil {
		return nil, err
	}
	sub := g.Data.EmptyGraph()
	sub.AddNode(u)
	for _, goal := range goals {
		v, err := g.NodeForErr(goal)
		if err != nil {
			return nil, err
		}
		// Find shortest paths, and shortest+1 paths
		paths := path.YenKShortestPaths(g, -1, 1, u, v)
		for _, path := range paths {
			for i := 1; i < len(path); i++ {
				lines := g.Lines(path[i-1].ID(), path[i].ID())
				for lines.Next() {
					sub.SetLine(lines.Line())
				}
			}
		}
	}
	return sub, nil
}

// Neighbors returns a breadth-first neighborhood following up to maxDepth edges from start.
func (g *Graph) Neighbors(start korrel8r.Class, maxDepth int) (*Graph, error) {
	sub := g.Data.EmptyGraph()
	u, err := g.NodeForErr(start)
	if err != nil {
		return nil, err
	}
	sub.AddNode(u)
	depth := 0
	bf := traverse.BreadthFirst{
		Traverse: func(e graph.Edge) bool {
			ok := depth < maxDepth
			if ok {
				EdgeFor(e).EachLine(func(l *Line) { sub.SetLine(l) })
			}
			return ok
		}}
	_ = bf.Walk(g, u, func(n graph.Node, d int) bool { depth = d; return d > maxDepth })
	return sub, nil
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

// Remove empty nodes and lines from the graph.
func (g *Graph) RemoveEmpty() {
	g.EachLine(func(l *Line) {
		if l.Queries.Total() == 0 {
			g.RemoveLine(l.F.ID(), l.T.ID(), l.ID())
		}
	})
	g.EachNode(func(n *Node) {
		if len(n.Result.List()) == 0 {
			g.RemoveNode(n.ID())
		}
	})
}

func (g *Graph) LineStrings() (lines []string) {
	g.EachLine(func(l *Line) { lines = append(lines, l.String()) })
	slices.Sort(lines)
	return lines
}

func (g *Graph) NodeStrings(sorted bool) (nodes []string) {
	g.EachNode(func(n *Node) { nodes = append(nodes, n.String(sorted)) })
	slices.Sort(nodes)
	return nodes
}

// Pull useful functions into this package.
var (
	NodesOf = graph.NodesOf
	LinesOf = graph.LinesOf
	EdgesOf = graph.EdgesOf
)
