// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package graph separates shared rule definitions from mutable search results.
//
// Data owns class/rule identity and compact integer topology. Node IDs identify
// classes; line IDs identify expanded (start class, goal class, rule) records.
// A Graph contains Node and Line objects for Gonum algorithms, rendering or
// results. It may contain only a subset of Data's topology. An Edge groups all
// parallel lines between two nodes; it has no separate identity in Data.
//
// Searches can use Data's ID-based adjacency without constructing a Graph.
// Mutable Nodes, Lines and Queries belong to a search/result, not to Data.
// View adapts Data directly to Gonum algorithms without result-bearing objects.
// This package does not query stores; the engine does that.
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
)

// Graph is a Gonum directed multigraph built from a Data definition.
// Nodes and Lines carry results, query accounting and GraphViz attributes. Graphs
// sharing Data use the same IDs, but normally own independent mutable objects.
//
// Graph is not safe for concurrent mutation. For concurrent read-only topology
// access, use Data.Graph's View instead. Use AddLine (not the embedded SetLine)
// to maintain the ordered line index.
type Graph struct {
	*multi.DirectedGraph
	GraphAttrs, NodeAttrs, EdgeAttrs Attrs
	Data                             *Data
	allLines                         []*Line // Insertion-order index for allocation-free EachLine.
}

// New creates an empty mutable Graph using data's identity space.
// It does not add any nodes or lines.
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

// FullGraph creates an independent mutable Graph containing all of Data.
// Nodes and lines have fresh, empty result/query/attribute state.
func (d *Data) FullGraph() *Graph {
	g := New(d)
	for id := range d.classes {
		g.AddNode(d.NewNode(int64(id)))
	}
	for id := range d.topology.lines {
		from, to := d.Endpoints(id)
		g.AddLine(d.NewLine(id, g.Node(from).(*Node), g.Node(to).(*Node)))
	}
	return g
}

// Weight an edge by the "spread" of its rules.
//
// Wildcard rules in domains with many classes (e.g. k8s, DependentToOwner)
// create many speculative graph lines, but following these lines often leads nowhere.
// The weight of an edge is the spread of its least expensive rule.
// In other words consider an edge expensive if it only has expensive rules.
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

// NodeFor returns this Graph's node for c, or nil if it is not in this Graph.
// Data.NodeID instead looks up topology membership without creating a Node.
func (g *Graph) NodeFor(c korrel8r.Class) *Node {
	id, ok := g.Data.NodeID(c)
	if !ok {
		return nil
	}
	gn := g.Node(id)
	if gn == nil {
		return nil
	}
	return gn.(*Node)
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

// EachLine visits the graph's line objects in insertion order without allocating.
func (g *Graph) EachLine(visit func(*Line)) {
	for _, l := range g.allLines {
		visit(l)
	}
}

// copyNode returns the mutable copy of n in g, creating one if absent.
func (g *Graph) copyNode(n *Node) *Node {
	if existing := g.Node(n.ID()); existing != nil {
		return existing.(*Node)
	}
	c := n.Copy()
	g.AddNode(c)
	return c
}

// copyLine copies l into g with fresh mutable state, copying endpoint nodes if needed.
// Like Node.Copy and Line.Copy, this copies identity, not accumulated results.
func (g *Graph) copyLine(l *Line) *Line {
	c := l.Copy(g.copyNode(l.Start()), g.copyNode(l.Goal()))
	g.AddLine(c)
	return c
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

// Select creates a mutable sub-graph of all lines where keep(line) is true.
func (g *Graph) Select(keep func(*Line) bool) *Graph {
	sub := New(g.Data)
	g.EachLine(func(l *Line) {
		if keep(l) {
			sub.copyLine(l)
		}
	})
	return sub
}

func (g *Graph) DOTID() string { return g.GraphAttrs["name"] }
func (g *Graph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	return g.GraphAttrs, g.NodeAttrs, g.EdgeAttrs
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

// AddLine inserts a previously absent line into Gonum and the ordered line index.
// It retains l itself (including Queries), rather than copying it. Its endpoints
// must be the nodes owned by this Graph. Use copyLine for fresh mutable copies.
func (g *Graph) AddLine(l *Line) {
	g.SetLine(l)
	g.allLines = append(g.allLines, l)
}

// Remove empty nodes and lines from the graph.
func (g *Graph) RemoveEmpty() {
	kept := g.allLines[:0]
	for _, l := range g.allLines {
		if l.Queries.Total() == 0 {
			g.RemoveLine(l.F.ID(), l.T.ID(), l.ID())
		} else {
			kept = append(kept, l)
		}
	}
	g.allLines = kept
	g.EachNode(func(n *Node) {
		if len(n.Result.List()) == 0 {
			g.RemoveNode(n.ID())
		}
	})
}

// RemoveEmptyGoalPaths removes nodes not connected to a non-empty goal.
func (g *Graph) RemoveEmptyGoalPaths(goals []korrel8r.Class) {
	// Get non-empty goal node IDs
	var goalIDs []int64
	for _, c := range goals {
		n := g.NodeFor(c)
		if n != nil && !n.Empty() {
			goalIDs = append(goalIDs, n.ID())
		}
	}
	// Remove  nodes not connected to one of goalIDs
	paths := path.DijkstraAllPaths(g)
	g.EachNode(func(n *Node) {
		var connected bool
		for _, goalID := range goalIDs {
			p, _, _ := paths.Between(n.ID(), goalID)
			connected = connected || p != nil
		}
		if !connected {
			g.RemoveNode(n.ID())
		}
	})
	// Rebuild allLines — RemoveNode removes edges from gonum but not from allLines.
	kept := g.allLines[:0]
	for _, l := range g.allLines {
		if g.Node(l.F.ID()) != nil && g.Node(l.T.ID()) != nil {
			kept = append(kept, l)
		}
	}
	g.allLines = kept
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
