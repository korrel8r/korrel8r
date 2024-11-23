// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"cmp"
	"slices"
	"strings"

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
	if g == nil {
		return nil
	}
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
	if g == nil {
		return nil
	}
	g.EachEdge(func(e *graph.Edge) {
		if !e.Goal().Empty() { // Skip edges that lead to an empty node.
			edges = append(edges, edge(e, opts.Rules))
		}
	})
	return edges
}

// Normalize API values by sorting slices in a predictable order.
// Useful for tests that need to compare actual and expected results.
func Normalize(v any) any {
	switch v := v.(type) {
	case Graph:
		Normalize(v.Nodes)
		Normalize(v.Edges)
	case []Node:
		slices.SortFunc(v, func(a, b Node) int { return strings.Compare(a.Class, b.Class) })
		for _, n := range v {
			Normalize(n)
		}
	case []Edge:
		slices.SortFunc(v, func(a, b Edge) int {
			if n := strings.Compare(a.Start, b.Start); n != 0 {
				return n
			} else {
				return strings.Compare(a.Goal, b.Goal)
			}
		})
		for _, e := range v {
			Normalize(e)
		}
	case Node:
		Normalize(v.Queries)
	case Edge:
		for _, r := range v.Rules {
			Normalize(r.Queries)
		}
	case []QueryCount:
		slices.SortFunc(v, func(a, b QueryCount) int { return strings.Compare(a.Query, b.Query) })
	}
	return v
}
