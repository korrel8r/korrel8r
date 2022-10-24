package graph

import (
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/yourbasic/graph"
)

// Graph is a directed multigraph with korrel8.Class vertices and korrel8.Rule edges.
//
// Immutable: cannot be modified after created, allows safe concurrent use.
type Graph struct {
	vertexID map[korrel8.Class]int
	vertices []korrel8.Class
	adjacent [][]korrel8.Rule
	edges    map[edge][]korrel8.Rule
	graph    *graph.Immutable
}

type edge struct{ start, goal int }

func New() *Graph {
	g := &Graph{vertexID: map[korrel8.Class]int{}, edges: map[edge][]korrel8.Rule{}}
	return g
}

func (g *Graph) Add(rules []korrel8.Rule) {
	// Create all the vertices.
	for _, r := range rules {
		start := g.addVertex(r.Start())
		goal := g.addVertex(r.Goal())
		g.adjacent[start] = append(g.adjacent[start], r)
		e := edge{start, goal}
		g.edges[e] = append(g.edges[e], r)
	}
	g.graph = graph.Sort(iterator{g})
}

func (g *Graph) addVertex(vertex korrel8.Class) int {
	id, ok := g.vertexID[vertex]
	if !ok {
		id = len(g.vertices)
		g.vertexID[vertex] = id
		g.vertices = append(g.vertices, vertex)
		g.adjacent = append(g.adjacent, nil)
	}
	return id
}

type iterator struct{ *Graph } // To set up yourbasic/graph
func (g iterator) Order() int  { return len(g.vertices) }
func (g iterator) Visit(v int, do func(w int, c int64) (skip bool)) (aborted bool) {
	for _, r := range g.adjacent[v] {
		_ = do(g.vertexID[r.Goal()], 0)
	}
	return false
}

// ShortestPath returns a shortest edge path from start to goal.
func (g *Graph) ShortestPath(start, goal korrel8.Class) []korrel8.Rule {
	_, ok1 := g.vertexID[start]
	_, ok2 := g.vertexID[goal]
	if !(ok1 && ok2) {
		return nil
	}
	p, _ := graph.ShortestPath(g.graph, g.vertexID[start], g.vertexID[goal])
	if len(p) > 1 { // Need at least 2 vertices to have an edge
		path := make([]korrel8.Rule, len(p)-1)
		start := p[0]
		for i, goal := range p[1:] {
			e := edge{start, goal}
			if r := g.edges[e]; len(r) > 0 {
				path[i] = r[0] // TODO: Ignoring multigraph
			}
			start = goal
		}
		return path
	}
	return nil
}
