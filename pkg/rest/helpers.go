// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"cmp"
	"slices"

	"github.com/korrel8r/korrel8r/pkg/graph"
)

func queryCounts(gq graph.Queries) []QueryCount {
	qcs := make([]QueryCount, 0, len(gq))
	for _, qc := range gq {
		qcs = append(qcs, QueryCount{Query: qc.Query.String(), Count: qc.Count})
	}
	slices.SortFunc(qcs, func(a, b QueryCount) int {
		if n := cmp.Compare(a.Count, b.Count); n != 0 {
			return -n
		}
		return cmp.Compare(a.Query, b.Query)
	})
	return qcs
}

func rule(l *graph.Line) (r Rule) {
	r.Name = l.Rule.Name()
	r.Queries = queryCounts(l.Queries)
	return r
}

func node(n *graph.Node) Node {
	return Node{
		Class:   n.Class.String(),
		Queries: queryCounts(n.Queries),
		Count:   len(n.Result.List()),
	}
}

func nodes(g *graph.Graph) []Node {
	nodes := []Node{} // Want [] not null for empty in JSON.
	g.EachNode(func(n *graph.Node) {
		if !n.Empty() { // Skip empty nodes
			nodes = append(nodes, node(n))
		}
	})
	return nodes
}

func edge(e *graph.Edge, rules bool) Edge {
	edge := Edge{
		Start: e.Start().Class.String(),
		Goal:  e.Goal().Class.String(),
	}
	if rules {
		e.EachLine(func(l *graph.Line) {
			if l.Queries.Total() != 0 {
				edge.Rules = append(edge.Rules, rule(l))
			}
		})
	}
	return edge
}

func edges(g *graph.Graph, opts *Options) (edges []Edge) {
	g.EachEdge(func(e *graph.Edge) {
		if !e.Goal().Empty() { // Skip edges that lead to an empty node.
			edges = append(edges, edge(e, opts.Rules))
		}
	})
	return edges
}
