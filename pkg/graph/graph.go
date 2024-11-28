// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package graph provides a directed multi-graph with class nodes and rule edges.
package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
)

// Graph is a directed multigraph with [korrel8r.Class] noes and [korrel8r.Rule] lines.
// Nodes and lines carry attributes for rendering by GraphViz.
//
// Concurrency: Graph is mutable, normal concurrency rules apply regarding read/write operations.
type Graph struct {
	*multi.DirectedGraph
	GraphAttrs, NodeAttrs, EdgeAttrs Attrs
	Data                             *Data

	shortest *path.AllShortest
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

func (g *Graph) NodeFor(c korrel8r.Class) *Node { return g.Data.NodeFor(c) }
func (g *Graph) NodesFor(classes ...korrel8r.Class) (nodes []*Node) {
	for _, c := range classes {
		nodes = append(nodes, g.NodeFor(c))
	}
	return nodes
}

func (g *Graph) EachNode(visit func(*Node)) {
	nodes := g.Nodes()
	for nodes.Next() {
		visit(nodes.Node().(*Node))
	}
}

func (g *Graph) AllNodes() (nodes []*Node) {
	g.EachNode(func(n *Node) { nodes = append(nodes, n) })
	return nodes
}

func (g *Graph) EachEdge(visit func(*Edge)) {
	edges := g.Edges()
	for edges.Next() {
		edge := Edge(edges.Edge().(multi.Edge))
		visit(&edge)
	}
}

func (g *Graph) EachLine(visit func(*Line)) {
	g.EachEdge(func(e *Edge) { e.EachLine(visit) })
}

func (g *Graph) AllLines() (lines []*Line) {
	g.EachLine(func(n *Line) { lines = append(lines, n) })
	return lines
}

func (g *Graph) EachLineTo(v *Node, visit func(*Line)) {
	u := g.To(v.ID())
	for u.Next() {
		l := g.Lines(u.Node().ID(), v.ID())
		for l.Next() {
			visit(l.Line().(*Line))
		}
	}
}

func (g *Graph) EachLineFrom(u *Node, visit func(*Line)) {
	v := g.From(u.ID())
	for v.Next() {
		l := g.Lines(u.ID(), v.Node().ID())
		for l.Next() {
			visit(l.Line().(*Line))
		}
	}
}

func (g *Graph) LinesBetween(u, v *Node) (lines []*Line) {
	l := g.Lines(u.ID(), v.ID())
	for l.Next() {
		lines = append(lines, l.Line().(*Line))
	}
	return lines
}

// newPaths returns a sub-graph of g containing only lines on the paths.
func (g *Graph) newPaths(paths [][]graph.Node) *Graph {
	sub := g.Data.EmptyGraph()
	for _, path := range paths {
		for i := 1; i < len(path); i++ {
			lines := g.Lines(path[i-1].ID(), path[i].ID())
			for lines.Next() {
				sub.SetLine(lines.Line())
			}
		}
	}
	return sub
}

// NodesSubgraph returns a new graph containing nodes and all lines between them.
func (g *Graph) NodesSubgraph(nodes []graph.Node) *Graph {
	sub := g.Data.EmptyGraph()
	nodeSet := unique.Set[int64]{}
	for _, n := range nodes {
		nodeSet.Add(n.ID())
	}
	for _, n := range nodes {
		to := g.From(n.ID())
		for to.Next() {
			if nodeSet.Has(to.Node().ID()) {
				lines := g.Lines(n.ID(), to.Node().ID())
				for lines.Next() {
					sub.SetLine(lines.Line())
				}
			}
		}
	}
	return sub
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
	if g.shortest == nil {
		shortest := path.DijkstraAllPaths(g)
		g.shortest = &shortest
	}
	var paths [][]graph.Node
	for _, goal := range goals {
		p, _ := g.shortest.AllBetween(g.NodeFor(start).ID(), g.NodeFor(goal).ID())
		paths = append(paths, p...)
	}
	return g.newPaths(paths)
}

// AllPaths returns a new sub-graph containing all paths between start and goal.
func (g *Graph) AllPaths(start korrel8r.Class, goals ...korrel8r.Class) *Graph {
	u := g.NodeFor(start)
	ap := allPaths{
		g:       g,
		visited: map[int64]bool{u.ID(): true},
		path:    []graph.Node{u},
		v:       unique.Set[int64]{},
	}
	for _, goal := range goals {
		ap.v.Add(g.NodeFor(goal).ID())
	}
	ap.run(u.ID())
	return g.newPaths(ap.paths)
}

// allPaths is the state of a backtracking depth-first-search
type allPaths struct {
	g       graph.Graph
	visited map[int64]bool
	path    []graph.Node
	paths   [][]graph.Node
	v       unique.Set[int64]
}

func (ap *allPaths) run(u int64) {
	iter := ap.g.From(u)
	for iter.Next() {
		n := iter.Node()
		if ap.visited[n.ID()] {
			continue
		}
		ap.path = append(ap.path, n)
		if ap.v.Has(n.ID()) { // Found a path
			path := make([]graph.Node, len(ap.path))
			copy(path, ap.path)
			ap.paths = append(ap.paths, path)
		}
		ap.visited[n.ID()] = true
		ap.run(n.ID())
		ap.visited[n.ID()] = false
		ap.path = ap.path[0 : len(ap.path)-1] // Backtrack and continue search.
	}
}
