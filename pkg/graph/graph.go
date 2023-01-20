package graph

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/multi"
	"gonum.org/v1/gonum/graph/path"
)

// Graph is a directed multigraph with korrel8r.Class vertices and korrel8r.Rule edges.
type Graph struct {
	name string
	*multi.DirectedGraph
	rules   []korrel8r.Rule
	classes []korrel8r.Class
	nodeID  map[korrel8r.Class]int64
	paths   path.AllShortest
}

// Line is a graph edge, corresponds to a rule.
type Line struct {
	multi.Line
	korrel8r.Rule
	Attrs map[string]string
}

func attrs(m map[string]string) (attrs []encoding.Attribute) {
	for k, v := range m {
		attrs = append(attrs, encoding.Attribute{Key: k, Value: v})
	}
	return attrs
}

func (l *Line) DOTID() string                    { return l.Rule.String() }
func (l *Line) Attributes() []encoding.Attribute { return attrs(l.Attrs) }

// Node is a graph Node, corresponds to a Class.
type Node struct {
	multi.Node
	korrel8r.Class
	Attrs map[string]string
}

func (n *Node) DOTID() string                    { return korrel8r.ClassName(n.Class) }
func (n *Node) Attributes() []encoding.Attribute { return attrs(n.Attrs) }

// New graph: nodes are classes, rules are edges from start to goal.
func New(name string, rules []korrel8r.Rule, extra []korrel8r.Class) *Graph {
	g := &Graph{name: name, DirectedGraph: multi.NewDirectedGraph(),
		rules:  rules,
		nodeID: map[korrel8r.Class]int64{},
	}
	for i, r := range g.rules {
		f, t := g.NodeForClass(r.Start()), g.NodeForClass(r.Goal())
		g.SetLine(&Line{Line: multi.Line{F: f, T: t, UID: int64(i)}, Rule: r, Attrs: map[string]string{"tooltip": r.String()}})
	}
	for _, c := range extra { // Extra classes
		g.NodeForClass(c)
	}
	g.paths = path.DijkstraAllPaths(g.DirectedGraph)
	return g
}

func (g *Graph) Name() string { return g.name }

// NodeForClass returns the node for class c, creating it if necessary.
func (g *Graph) NodeForClass(c korrel8r.Class) *Node {
	id, ok := g.nodeID[c]
	if !ok {
		id = int64(len(g.classes))
		g.classes = append(g.classes, c)
		g.nodeID[c] = id
		n, _ := g.NodeWithID(id)
		g.AddNode(&Node{Node: n.(multi.Node), Class: c, Attrs: map[string]string{}})
	}
	return g.Node(id).(*Node)
}

func (g *Graph) findNodeID(c korrel8r.Class) (int64, error) {
	if id, ok := g.nodeID[c]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("class not found in graph: %v", c)
}

type pathFunc func(u, v int64) (paths [][]graph.Node)

func (g *Graph) multiPaths(start, goal korrel8r.Class, f pathFunc) (mp []MultiPath, err error) {
	if start == goal {
		return nil, fmt.Errorf("same start and goal class: a%v", start)
	}
	var startID, goalID int64
	if startID, err = g.findNodeID(start); err != nil {
		return nil, err
	}
	if goalID, err = g.findNodeID(goal); err != nil {
		return nil, err
	}
	for _, path := range f(startID, goalID) {
		mp = append(mp, newMultiPath(g, path))
	}
	return mp, nil
}

// ShortestPaths returns all the shortest paths from start to goal.
func (g *Graph) ShortestPaths(start, goal korrel8r.Class) ([]MultiPath, error) {
	return g.multiPaths(start, goal, func(u, v int64) [][]graph.Node {
		paths, _ := g.paths.AllBetween(u, v)
		return paths
	})
}

// KPaths returns the k-shortest paths from start to goal.
func (g *Graph) KShortestPaths(start, goal korrel8r.Class, k int) ([]MultiPath, error) {
	return g.multiPaths(start, goal, func(u, v int64) [][]graph.Node {
		return path.YenKShortestPaths(g, k, g.Node(u), g.Node(v))
	})
}

func (g *Graph) AllPaths(start, goal korrel8r.Class) ([]MultiPath, error) {
	return g.multiPaths(start, goal, func(u, v int64) [][]graph.Node {
		return AllPaths(g, u, v)
	})
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
			e = append(e, lines.Line().(*Line).Rule)
		}
		mp = append(mp, e)
	}
	return mp
}
