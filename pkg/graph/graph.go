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
	Data                             *Data
	GraphAttrs, NodeAttrs, EdgeAttrs Attrs
	shortest                         *path.AllShortest
}

// New empty graph based on Data
func New(data *Data) *Graph {
	return &Graph{
		DirectedGraph: multi.NewDirectedGraph(),
		Data:          data,
		GraphAttrs: Attrs{
			"fontname": "Helvetica",
			"fontsize": "12",
			"center":   "true",
			"splines":  "spline",
			"overlap":  "false",
			"sep":      "+8",
		},
		NodeAttrs: Attrs{
			"fontname": "Helvetica",
			"fontsize": "12",
			"shape":    "box",
		},
		EdgeAttrs: Attrs{
			"fontname": "Helvetica",
			"fontsize": "12",
		},
	}
}

func (g *Graph) NodeFor(c korrel8r.Class) *Node { return g.Data.NodeFor(c) }

func (g *Graph) EachLine(visit func(*Line)) {
	edges := g.Edges()
	for edges.Next() {
		lines := edges.Edge().(multi.Edge)
		for lines.Next() {
			visit(lines.Line().(*Line))
		}
	}
}

func (g *Graph) AllLines() []*Line {
	var lines []*Line
	g.EachLine(func(l *Line) { lines = append(lines, l) })
	return lines
}

func (g *Graph) EachNode(visit func(*Node)) {
	nodes := g.Nodes()
	for nodes.Next() {
		visit(nodes.Node().(*Node))
	}
}

func (g *Graph) AllNodes() []*Node {
	var nodes []*Node
	g.EachNode(func(n *Node) { nodes = append(nodes, n) })
	return nodes
}

func (g *Graph) EachLineTo(v *Node, traverse func(*Line)) {
	u := g.To(v.ID())
	for u.Next() {
		l := g.Lines(u.Node().ID(), v.ID())
		for l.Next() {
			traverse(l.Line().(*Line))
		}
	}
}

func (g *Graph) LinesTo(v *Node) (lines []*Line) {
	g.EachLineTo(v, func(l *Line) { lines = append(lines, l) })
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
func (g *Graph) Select(keep func(l *Line) bool) *Graph {
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

// ShortestPaths returns a new sub-graph containing all shortest paths between start and goal.
func (g *Graph) ShortestPaths(start, goal korrel8r.Class) *Graph {
	if g.shortest == nil {
		shortest := path.DijkstraAllPaths(g)
		g.shortest = &shortest
	}
	paths, _ := g.shortest.AllBetween(g.NodeFor(start).ID(), g.NodeFor(goal).ID())
	return g.newPaths(paths)
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
	return g.newPaths(ap.paths)
}

// allPaths is the state of a backtracking depth-first-search
type allPaths struct {
	g       graph.Graph
	visited map[int64]bool
	path    []graph.Node
	paths   [][]graph.Node
}

func (ap *allPaths) run(u, v int64) {
	iter := ap.g.From(u)
	for iter.Next() {
		n := iter.Node()
		if ap.visited[n.ID()] {
			continue
		}
		ap.path = append(ap.path, n)
		if n.ID() == v { // Complete path
			path := make([]graph.Node, len(ap.path))
			copy(path, ap.path)
			ap.paths = append(ap.paths, path)
		} else { // Continue search
			ap.visited[n.ID()] = true
			ap.run(n.ID(), v)
			ap.visited[n.ID()] = false
		}
		ap.path = ap.path[0 : len(ap.path)-1] // Backtrack and continue search.
	}
}
