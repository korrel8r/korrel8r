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
)

type ruleMap map[string]korrel8r.Rule

func (rm ruleMap) r(i, j int) korrel8r.Rule {
	name := fmt.Sprintf("%v_%v", i, j)
	if _, ok := rm[name]; !ok {
		rm[name] = rules.NewTemplateRule([]korrel8r.Class{c(i)}, []korrel8r.Class{c(j)}, template.New(name))
	}
	return rm[name]
}

func TestGraph_Select(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	for _, x := range []struct {
		name        string
		graph, want []korrel8r.Rule
		pick        func(*Line) bool
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

func TestGraph_LinesBetween(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }
	g := testGraph([]rule{r(1, 2), r(1, 3), r(1, 2), r(12, 13)})
	rules := func(start, goal korrel8r.Class) []string {
		var rules []string
		g.EachLineBetween(g.NodeFor(start), g.NodeFor(goal), func(l *Line) {
			rules = append(rules, l.Rule.Name())
		})
		return rules
	}
	assert.ElementsMatch(t, rules(c(1), c(2)), []string{"1_2", "1_2"})
	assert.ElementsMatch(t, rules(c(1), c(3)), []string{"1_3"})
	assert.Empty(t, rules(c(1), c(13)))
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
