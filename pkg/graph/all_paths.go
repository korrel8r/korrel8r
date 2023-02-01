package graph

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph"
)

// AllPathsGraph returns a sub-graph of rules that are on any path from u to v
func AllPathsGraph(g *Graph, u, v korrel8r.Class) *Graph {
	paths := AllPaths(g, g.NodeForClass(u).ID(), g.NodeForClass(v).ID())
	// Collect all rules that are on any path
	var rules []korrel8r.Rule
	for _, path := range paths {
		for i := 0; i < len(path)-1; i++ {
			if lines := g.Lines(path[i].ID(), path[i+1].ID()); lines != nil {
				for lines.Next() {
					rules = append(rules, lines.Line().(*Line).Rule)
				}
			}
		}
	}
	name := fmt.Sprintf("Paths %v -> %v", korrel8r.ClassName(u), korrel8r.ClassName(v))
	return New(name, rules)
}

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
