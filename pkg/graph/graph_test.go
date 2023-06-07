// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/graph"
)

func TestNodesSubGraph(t *testing.T) {
	for _, x := range []struct {
		name        string
		graph, want []rule
		include     []int
	}{
		{
			name:    "one",
			graph:   []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1, 3, 12},
			want:    []rule{r(1, 3), r(3, 12)},
		},
		{
			name:    "two",
			graph:   []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1},
			want:    nil,
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var nodes []graph.Node
			for _, i := range x.include {
				nodes = append(nodes, g.NodeFor(c(i)))
			}
			sub := g.NodesSubgraph(nodes)
			assert.Equal(t, x.want, graphRules(sub))
		})
	}
}

func TestGraph_NodesSubGraph(t *testing.T) {
	for _, x := range []struct {
		name        string
		graph, want []rule
		include     []int
	}{
		{
			name:    "one",
			graph:   []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1, 3, 12},
			want:    []rule{r(1, 3), r(3, 12)},
		},
		{
			name:    "two",
			graph:   []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1},
			want:    nil,
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var nodes []graph.Node
			for _, i := range x.include {
				nodes = append(nodes, g.NodeFor(c(i)))
			}
			sub := g.NodesSubgraph(nodes)
			assert.Equal(t, x.want, graphRules(sub))
		})
	}
}

func TestGraph_Select(t *testing.T) {
	for _, x := range []struct {
		name        string
		graph, want []rule
		pick        func(l *Line) bool
	}{
		{
			name:  "one",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			pick:  func(l *Line) bool { return unique.Set[rule]{r(1, 3): {}, r(3, 11): {}}.Has(l.Rule) },
			want:  []rule{r(1, 3), r(3, 11)},
		},
		{
			name:  "two",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			pick:  func(l *Line) bool { return false },
			want:  nil,
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph).Select(x.pick)
			assert.Equal(t, x.want, graphRules(g))
		})
	}
}
