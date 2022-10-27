package graph

import (
	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
)

var log = logging.Log

// Graph is a directed multigraph with korrel8.Class vertices and korrel8.Rule edges.
type Graph struct {
	*multi.DirectedGraph
	rules   []korrel8.Rule
	classes []korrel8.Class
	nodeID  map[korrel8.Class]int64
	paths   path.AllShortest
}

// New graph: nodes are classes, edge for each rule is (rule.start, rule.goal)
func New(rules []korrel8.Rule, classes []korrel8.Class) *Graph {
	g := &Graph{DirectedGraph: multi.NewDirectedGraph(),
		rules:   rules,
		classes: classes,
		nodeID:  map[korrel8.Class]int64{},
	}
	for i, r := range g.rules {
		f, t := g.addClass(r.Start()), g.addClass(r.Goal())
		g.SetLine(line{Line: multi.Line{F: f, T: t, UID: int64(i)}, Rule: r})
	}
	for _, c := range g.classes { // Extra classes
		g.addClass(c)
	}
	// FIXME expand wildcards here.
	g.paths = path.DijkstraAllPaths(g.DirectedGraph)
	return g
}

func (g *Graph) addClass(c korrel8.Class) graph.Node {
	if _, ok := g.nodeID[c]; !ok {
		id := int64(len(g.classes))
		n, _ := g.NodeWithID(id)
		g.AddNode(node{Node: n.(multi.Node), Class: c})
		g.nodeID[c] = id
		g.classes = append(g.classes, c)
	}
	return g.Node(g.nodeID[c])
}

// node for a class
func (g *Graph) node(c korrel8.Class) graph.Node {
	return g.Node(g.nodeID[c])
}

// FIXME multi-path!?
func (g *Graph) ShortestPath(start, goal korrel8.Class) []korrel8.Rule {
	if start == goal {
		return nil
	}
	p, _, multi := g.paths.Between(g.nodeID[start], g.nodeID[goal])
	if multi {
		log.V(3).Info("multiple paths from %v to %v", start, goal)
	}
	rules := []korrel8.Rule{}
	u := p[0]
	for _, v := range p[1:] {
		lines := g.Lines(u.ID(), v.ID())
		if lines.Len() > 1 {
			log.V(3).Info("multiple lines from %v to %v", start, goal)
		}
		lines.Next()
		l := lines.Line()
		r := l.(line).Rule
		rules = append(rules, r)
		u = v
	}
	return rules
}

func (g *Graph) DOTAttributers() (graph, node, edge encoding.Attributer) {
	return nil,
		&encoding.Attributes{
			{Key: "fontname", Value: "Helvetica"},
			{Key: "shape", Value: "box"},
			{Key: "style", Value: "rounded,filled"},
			{Key: "fillcolor", Value: "cyan"},
			{Key: "margin", Value: ".1"},
			{Key: "width", Value: ".1"},
			{Key: "height", Value: ".1"},
		},
		&encoding.Attributes{
			{Key: "fontname", Value: "Helvetica-Oblique"},
			{Key: "fontsize", Value: "10"},
		}
}

// line is a graph edge, contains a rule.
type line struct {
	multi.Line
	korrel8.Rule
}

func (l line) DOTID() string { return l.String() }
func (l line) Attributes() []encoding.Attribute {
	return []encoding.Attribute{
		{Key: "tooltip", Value: l.String()},
	}
}

// node is a graph node, contains a Class.
type node struct {
	multi.Node
	korrel8.Class
}

func (n node) DOTID() string { return n.String() }
func (n node) Attributes() []encoding.Attribute {
	return []encoding.Attribute{}
}
