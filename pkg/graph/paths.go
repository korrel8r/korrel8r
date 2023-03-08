package graph

import (
	"gonum.org/v1/gonum/graph"
)

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

// visitPath vists each rule connecting nodes in path.
func visitPath(g *Graph, path []graph.Node, visit func(Edge)) {
	for i := 1; i < len(path); i++ {
		visit(WrapEdge(g.Edge(path[i-1].ID(), path[i].ID())))
	}
}

// visitPath vists each rule connecting nodes in path.
func visitPaths(g *Graph, paths [][]graph.Node, visit func(Edge)) {
	for _, path := range paths {
		visitPath(g, path, visit)
	}
}

func visitTo(g *Graph, to graph.Node, visit func(Edge)) {
	in := g.To(to.ID())
	for in.Next() {
		visit(WrapEdge(g.Edge(in.Node().ID(), to.ID())))
	}
}
