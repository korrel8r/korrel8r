package graph

import (
	"gonum.org/v1/gonum/graph"
)

// AllPaths returns all simple paths fro u to v.
func AllPaths(g graph.Graph, u, v int64) [][]graph.Node {
	ap := allPaths{
		g:       g,
		visited: map[int64]bool{u: true},
		path:    []graph.Node{g.Node(u)},
	}
	ap.run(u, v)
	return ap.paths
}

// dfs is the state of a backtracking depth-first-search
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
