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
	rules    unique.List[korrel8r.Rule]
	shortest *path.AllShortest
}

// Node is a graph Node, corresponds to a Class.
type Node struct {
	multi.Node
	Attrs  // GraphViz Attributer
	Class  korrel8r.Class
	Result *Result // Results accumulated from incoming lines
}

func ClassForNode(n graph.Node) korrel8r.Class { return n.(*Node).Class }

func (n *Node) String() string { return korrel8r.ClassName(n.Class) }
func (n *Node) DOTID() string  { return n.String() }

// Line is one line in a multi-graph edge, corresponds to a rule.
type Line struct {
	multi.Line
	Attrs  // GraphViz Attributer
	Rule   korrel8r.Rule
	Result *Result // Result of applying this rule while traversing the graph.
}

func (l *Line) DOTID() string { return l.Rule.String() }

// New graph
func New(rules ...korrel8r.Rule) *Graph {
	g := &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
		nodeID:        map[korrel8r.Class]int64{},
		rules:         unique.NewList[korrel8r.Rule](),
		GraphAttrs:    Attrs{},
		NodeAttrs: Attrs{
			"fontname": "Helvetica",
			"shape":    "box",
			"style":    "rounded,filled",
		},
		EdgeAttrs: Attrs{},
	}
	for _, r := range rules {
		g.AddRule(r)
	}
	return g
}

// AddRule adds a rule and its start/goal classes to the graph.
func (g *Graph) AddRule(r korrel8r.Rule) *Line {
	g.shortest = nil // Invalidated
	line := &Line{
		Line:   g.NewLine(g.NodeForClass(r.Start()), g.NodeForClass(r.Goal())).(multi.Line),
		Result: NewResult(r.Goal()),
		Rule:   r,
		Attrs:  Attrs{},
	}
	if g.rules.Add(r) {
		g.DirectedGraph.SetLine(line)
	}
	return line
}

// SetLine adds the rule from l to the graph, l must be *Line.
func (g *Graph) SetLine(l graph.Line) { g.AddRule(l.(*Line).Rule) }

// NodeForClass returns the node for class c, creating it if necessary.
func (g *Graph) NodeForClass(c korrel8r.Class) *Node {
	id, ok := g.nodeID[c]
	if !ok {
		n := g.NewNode()
		id = n.ID()
		g.nodeID[c] = id
		g.AddNode(&Node{Node: n.(multi.Node), Class: c, Attrs: Attrs{}, Result: NewResult(c)})
	}
	return g.Node(id).(*Node)
}

// ClassID returns node ID for class, or -1 if not found.
func (g *Graph) ClassID(c korrel8r.Class) int64 {
	if id, ok := g.nodeID[c]; ok {
		return id
	}
	return -1
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

// SubGraph constructs the sub-graph containing nodes and all edges between them.
func (g *Graph) SubGraph(nodes []graph.Node) *Graph {
	sub := New()
	for _, u := range nodes {
		for _, v := range nodes {
			lines := g.Lines(u.ID(), v.ID())
			for lines.Next() {
				sub.AddRule(lines.Line().(*Line).Rule)
			}
		}
	}
	return sub
}

func (g *Graph) PathGraph(paths [][]graph.Node) *Graph {
	sub := New()
	visitPaths(g, paths, func(e multi.Edge) {
		visitLines(e, func(l graph.Line) {
			sub.AddRule(l.(*Line).Rule)
		})
	})
	return sub
}

func (g *Graph) ShortestPaths(u, v korrel8r.Class) [][]graph.Node {
	if g.shortest == nil {
		shortest := path.DijkstraAllPaths(g)
		g.shortest = &shortest
	}
	paths, _ := g.shortest.AllBetween(g.NodeForClass(u).ID(), g.NodeForClass(v).ID())
	return paths
}

func (g *Graph) Clone() *Graph { return New(g.rules.List...) }

func (g *Graph) LinesTo(c korrel8r.Class) (lines []graph.Line) {
	cid := g.NodeForClass(c).ID()
	preds := g.To(cid)
	for preds.Next() {
		ls := g.Lines(preds.Node().ID(), cid)
		lines = append(lines, ls.(graph.LineSlicer).LineSlice()...)
	}
	return lines
}
