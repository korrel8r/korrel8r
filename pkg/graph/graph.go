package graph

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
)

// Graph is a directed multigraph with korrel8r.Class noes and korrel8r.Rule lines.
// Nodes and lines carry attributes for rendering by GraphViz.
type Graph struct {
	*multi.DirectedGraph
	GraphAttrs, NodeAttrs, EdgeAttrs Attrs

	nodeID   map[korrel8r.Class]int64
	shortest *path.AllShortest
}

// Node is a graph Node, corresponds to a Class.
type Node struct {
	graph.Node
	Attrs // GraphViz Attributer
	Class korrel8r.Class
}

func ClassForNode(n graph.Node) korrel8r.Class { return n.(*Node).Class }

func (n *Node) String() string { return korrel8r.ClassName(n.Class) }
func (n *Node) DOTID() string  { return n.String() }

// Line is one line in a multi-graph edge, corresponds to a rule.
type Line struct {
	graph.Line
	Attrs // GraphViz Attributer
	Rule  korrel8r.Rule
}

func (l *Line) DOTID() string            { return l.Rule.String() }
func RuleFor(l graph.Line) korrel8r.Rule { return l.(*Line).Rule }

// Edge is a type-safe wrapper for a multi.Edge.
type Edge struct{ multi.Edge }

// WrapEdge wraps an edge and resets its iterator.
func WrapEdge(e graph.Edge) Edge {
	me := e.(multi.Edge)
	me.Reset()
	return Edge{me}
}

func (e Edge) Start() korrel8r.Class { return ClassForNode(e.From()) }
func (e Edge) Goal() korrel8r.Class  { return ClassForNode(e.To()) }
func (e Edge) Line() *Line           { return e.Edge.Line().(*Line) }

// New graph is immutable.
func New(rules []korrel8r.Rule) *Graph {
	g := &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
		nodeID:        map[korrel8r.Class]int64{},
		GraphAttrs:    Attrs{},
		NodeAttrs: Attrs{
			"fontname": "Helvetica",
			"shape":    "box",
			"style":    "rounded,filled",
		},
		EdgeAttrs: Attrs{},
	}
	for _, r := range rules {
		g.addRule(r)
	}
	return g
}

// addRule adds a rule and its start/goal classes to the graph.
func (g *Graph) addRule(r korrel8r.Rule) {
	g.shortest = nil // Invalidated
	g.DirectedGraph.SetLine(&Line{
		Line:  g.NewLine(g.NodeFor(r.Start()), g.NodeFor(r.Goal())),
		Rule:  r,
		Attrs: Attrs{},
	})
}

func (g *Graph) EachEdge(visit func(Edge)) {
	edges := g.Edges()
	for edges.Next() {
		visit(WrapEdge(edges.Edge()))
	}
}

func (g *Graph) EachLine(visit func(*Line)) {
	g.EachEdge(func(e Edge) {
		for e.Next() {
			visit(e.Line())
		}
	})
}

func (g *Graph) EachNode(visit func(*Node)) {
	nodes := g.Nodes()
	for nodes.Next() {
		visit(nodes.Node().(*Node))
	}
}

// NodeFor returns the Node for class c, creating it if necessary.
func (g *Graph) NodeFor(c korrel8r.Class) *Node {
	id, ok := g.nodeID[c]
	if !ok {
		n := g.NewNode()
		id = n.ID()
		g.nodeID[c] = id
		g.AddNode(&Node{Node: n, Class: c, Attrs: Attrs{}})
	}
	return g.Node(id).(*Node)
}

func (g *Graph) DOTID() string { return g.GraphAttrs["name"] }

func (g *Graph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	return g.GraphAttrs, g.NodeAttrs, g.EdgeAttrs
}

// Attributes for nodes and lines rendered by Graphviz.
type Attrs map[string]string

var (
	_ encoding.Attributer = Attrs{}
	_ encoding.Attributer = Node{}
	_ encoding.Attributer = Line{}
)

func (a Attrs) Attributes() (enc []encoding.Attribute) {
	for k, v := range a {
		enc = append(enc, encoding.Attribute{Key: k, Value: v})
	}
	return enc
}

// SubGraph new graph of rules from lines where keep() returns true
func (g *Graph) SubGraph(keep func(*Line) bool) *Graph {
	sub := New(nil)
	g.EachLine(func(l *Line) {
		if keep(l) {
			sub.addRule(l.Rule)
		}
	})
	return sub
}

// subGraphOf constructs the sub-graph containing nodes and all edges between them.
func (g *Graph) subGraphOf(nodes []graph.Node) *Graph {
	sub := New(nil)
	for _, u := range nodes {
		for _, v := range nodes {
			if u != v {
				lines := g.Lines(u.ID(), v.ID())
				for lines.Next() {
					sub.addRule(lines.Line().(*Line).Rule)
				}
			}
		}
	}
	return sub
}

type edgeID [2]int64

func (g *Graph) pathGraph(paths [][]graph.Node) *Graph {
	sub := New(nil)
	seen := unique.Set[[2]int64]{}
	visitPaths(g, paths, func(e Edge) {
		if seen.Add(edgeID{e.F.ID(), e.T.ID()}) { // Added new
			for e.Next() {
				sub.addRule(e.Line().Rule)
			}
		}
	})
	return sub
}

// AllPaths returns a new sub-graph containing all paths between start and goal.
func (g *Graph) AllPaths(start, goal korrel8r.Class) *Graph {
	u, v := g.NodeFor(start), g.NodeFor(goal)
	ap := allPaths{
		g:       g,
		visited: map[int64]bool{u.ID(): true},
		path:    []graph.Node{u},
	}
	ap.run(u.ID(), v.ID())
	return g.pathGraph(ap.paths)
}

// ShortestPaths returns a new sub-graph containing all paths between start and goal.
func (g *Graph) ShortestPaths(start, goal korrel8r.Class) *Graph {
	if g.shortest == nil {
		shortest := path.DijkstraAllPaths(g)
		g.shortest = &shortest
	}
	paths, _ := g.shortest.AllBetween(g.NodeFor(start).ID(), g.NodeFor(goal).ID())
	return g.pathGraph(paths)
}

func (g *Graph) Clone() *Graph {
	g2 := New(nil)
	g.EachLine(func(l *Line) { g2.addRule(l.Rule) })
	return g2
}
