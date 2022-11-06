package graph

import (
	"fmt"

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
	name string
	*multi.DirectedGraph
	rules   []korrel8.Rule
	classes []korrel8.Class
	nodeID  map[korrel8.Class]int64
	paths   path.AllShortest
}

// line is a graph edge, corresponds to a rule.
type line struct {
	multi.Line
	korrel8.Rule
}

func (l line) DOTID() string { return "" }
func (l line) Attributes() []encoding.Attribute {
	return []encoding.Attribute{{Key: "tooltip", Value: l.String()}}
}

// node is a graph node, corresponds to a Class.
type node struct {
	multi.Node
	korrel8.Class
}

func (n node) DOTID() string                    { return korrel8.FullName(n.Class) }
func (n node) Attributes() []encoding.Attribute { return nil }

// New graph: nodes are classes, rules are edges from start to goal.
func New(name string, rules []korrel8.Rule, extra []korrel8.Class) *Graph {
	g := &Graph{name: name, DirectedGraph: multi.NewDirectedGraph(),
		rules:  rules,
		nodeID: map[korrel8.Class]int64{},
	}
	for i, r := range g.rules {
		f, t := g.addClass(r.Start()), g.addClass(r.Goal())
		g.SetLine(line{Line: multi.Line{F: f, T: t, UID: int64(i)}, Rule: r})
	}
	for _, c := range extra { // Extra classes
		g.addClass(c)
	}
	g.paths = path.DijkstraAllPaths(g.DirectedGraph)
	return g
}

func (g *Graph) Name() string { return g.name }

func (g *Graph) addClass(c korrel8.Class) graph.Node {
	id, ok := g.nodeID[c]
	if !ok {
		id = int64(len(g.classes))
		g.classes = append(g.classes, c)
		g.nodeID[c] = id
		n, _ := g.NodeWithID(id)
		g.AddNode(node{Node: n.(multi.Node), Class: c})
	}
	return g.Node(id)
}

// ShortestPaths returns all the shortest paths from start to goal.
func (g *Graph) ShortestPaths(start, goal korrel8.Class) ([]MultiPath, error) {
	var (
		startID, goalID int64
		ok              bool
	)
	if startID, ok = g.nodeID[start]; !ok {
		return nil, fmt.Errorf("start class not found in graph: %v", start)
	}
	if goalID, ok = g.nodeID[goal]; !ok {
		return nil, fmt.Errorf("goal class not found in graph: %v", goal)
	}
	if start == goal {
		return nil, fmt.Errorf("same start and goal class: a%v", start)
	}
	var mp []MultiPath
	paths, _ := g.paths.AllBetween(startID, goalID)
	for _, path := range paths {
		mp = append(mp, newMultiPath(g, path))
	}
	return mp, nil
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

func (g *Graph) DOTID() string { return g.Name() }

func newMultiPath(g graph.Multigraph, path []graph.Node) MultiPath {
	if len(path) == 0 {
		return nil
	}
	var mp MultiPath
	for i := 0; i < len(path)-1; i++ {
		var e Links
		lines := g.Lines(path[i].ID(), path[i+1].ID())
		for lines.Next() {
			e = append(e, lines.Line().(line).Rule)
		}
		mp = append(mp, e)
	}
	return mp
}
