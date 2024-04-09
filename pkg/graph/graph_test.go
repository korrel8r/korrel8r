// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"
	"text/template"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/graph"
)

type ruleMap map[string]korrel8r.Rule

func (rm ruleMap) r(i, j int) korrel8r.Rule {
	name := fmt.Sprintf("%v_%v", i, j)
	if _, ok := rm[name]; !ok {
		rm[name] = rules.NewTemplateRule([]korrel8r.Class{c(i)}, []korrel8r.Class{c(j)}, template.New(name))
	}
	return rm[name]
}

func TestGraph_NodesSubGraph(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	for _, x := range []struct {
		name        string
		graph, want []korrel8r.Rule
		include     []int
	}{
		{
			name:    "one",
			graph:   []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			include: []int{1, 3, 12},
			want:    []korrel8r.Rule{r(1, 3), r(3, 12)},
		},
		{
			name:    "two",
			graph:   []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
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
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	for _, x := range []struct {
		name        string
		graph, want []korrel8r.Rule
		pick        func(l *Line) bool
	}{
		{
			name:  "one",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			pick:  func(l *Line) bool { return unique.Set[korrel8r.Rule]{r(1, 3): {}, r(3, 11): {}}.Has(l.Rule) },
			want:  []korrel8r.Rule{r(1, 3), r(3, 11)},
		},
		{
			name:  "two",
			graph: []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
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

func TestGraph_EachNode(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) rule { return rm.r(i, j) }

	var nodes []*Node
	g := testGraph([]rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)})
	g.EachNode(func(n *Node) { nodes = append(nodes, n) })
	var want []*Node
	for _, i := range []int{1, 2, 3, 11, 12, 13} {
		want = append(want, g.NodeFor(c(i)))
	}
	assert.ElementsMatch(t, want, nodes)
}

func TestGraph_NodesBetween(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	g := testGraph([]rule{r(1, 2), r(1, 3), r(1, 2), r(12, 13)})
	assert.Len(t, g.LinesBetween(g.NodeFor(c(1)), g.NodeFor(c(2))), 2)
	assert.Len(t, g.LinesBetween(g.NodeFor(c(1)), g.NodeFor(c(3))), 1)
	assert.Len(t, g.LinesBetween(g.NodeFor(c(1)), g.NodeFor(c(13))), 0)
}

func TestGraph_LinesTo(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }
	g := testGraph([]rule{r(1, 2), r(1, 3), r(2, 3), r(12, 13)})
	assert.Len(t, g.LinesTo(g.NodeFor(c(1))), 0)
	assert.Len(t, g.LinesTo(g.NodeFor(c(2))), 1)
	assert.Len(t, g.LinesTo(g.NodeFor(c(3))), 2)
}

func TestGraph_AllPaths(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }
	for _, x := range []struct {
		name        string
		graph, want []rule
	}{
		{
			name:  "simple",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			want:  []rule{r(1, 3), r(3, 12), r(12, 13)},
		},
		{
			name:  "multiple",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(2, 12), r(3, 12), r(12, 13)},
			want:  []rule{r(1, 2), r(1, 3), r(2, 12), r(3, 12), r(12, 13)},
		},
		{
			name:  "lengths",
			graph: []rule{r(1, 2), r(1, 3), r(1, 13), r(2, 12), r(3, 12), r(12, 13)},
			want:  []rule{r(1, 2), r(1, 3), r(1, 13), r(2, 12), r(3, 12), r(12, 13)},
			// want:  [][]int{{1, 2, 12, 13}, {1, 3, 12, 13}, {1, 13}},
		},
		{
			name:  "empty",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(12, 13)},
			want:  nil,
		}} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			paths := g.AllPaths(c(1), c(13))
			mock.SortRules(x.want)
			assert.Equal(t, x.want, graphRules(paths))
		})
	}
}

func TestGraph_ShortestPaths(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }
	for _, x := range []struct {
		name        string
		graph, want []rule
	}{
		{
			name:  "simple",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			want:  []rule{r(1, 3), r(3, 12), r(12, 13)},
		},
		{
			name:  "multiple",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(2, 12), r(3, 12), r(12, 13)},
			want:  []rule{r(1, 2), r(1, 3), r(2, 12), r(3, 12), r(12, 13)},
		},
		{
			name:  "lengths",
			graph: []rule{r(1, 2), r(1, 3), r(1, 13), r(2, 12), r(3, 12), r(12, 13)},
			want:  []rule{r(1, 13)},
		},
		{
			name:  "empty",
			graph: []rule{r(1, 2), r(1, 3), r(3, 11), r(12, 13)},
			want:  nil,
		}} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			paths := g.ShortestPaths(c(1), c(13))
			mock.SortRules(x.want)
			assert.Equal(t, x.want, graphRules(paths))
		})
	}
}
