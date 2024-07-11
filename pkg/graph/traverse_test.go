// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
)

type collecter struct {
	rules   []string
	classes unique.Set[string]
}

func (c *collecter) Traverse(l *Line) bool {
	c.rules = append(c.rules, RuleFor(l).Name())
	if c.classes == nil {
		c.classes = unique.NewSet[string]()
	}
	c.classes.Add(ClassFor(l.From()).String())
	c.classes.Add(ClassFor(l.To()).String())
	return true
}

func TestTraverse(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) rule { return rm.r(i, j) }

	for _, x := range []struct {
		name    string
		graph   []rule
		rules   [][]string // inner slices are are unordered components.
		classes unique.Set[string]
	}{
		{
			name:    "multipath",
			graph:   []rule{r(1, 11), r(1, 12), r(11, 99), r(12, 99)},
			rules:   [][]string{{"1_11", "1_12"}, {"11_99", "12_99"}},
			classes: unique.NewSet("1", "11", "12", "99"),
		},
		{
			name:  "simple",
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 5)},
			rules: [][]string{{"1_2"}, {"2_3"}, {"3_4"}, {"4_5"}},
		},
		{
			name:  "cycle", // cycle of 2,3,4
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 2), r(4, 5)},
			rules: [][]string{{"1_2"}, {"2_3", "3_4", "4_2", "4_5"}},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var got collecter
			_, err := g.Traverse(x.graph[0].Start()[0], x.graph[len(x.graph)-1].Goal(), got.Traverse)
			assert.NoError(t, err)
			assertComponentOrder(t, x.rules, got.rules)
		})
	}
}

func TestNeighbours(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	g := testGraph([]rule{r(1, 11), r(11, 1), r(1, 12), r(1, 13), r(11, 22), r(12, 22), r(12, 13), r(22, 99)})
	for _, x := range []struct {
		depth int
		want  [][]string
	}{
		{
			depth: 1,
			want:  [][]string{{"1_11", "1_12", "1_13"}},
		},
		{
			depth: 2,
			want:  [][]string{{"1_11", "1_12", "1_13"}, {"11_22", "12_22"}},
		},
		{
			depth: 3,
			want:  [][]string{{"1_11", "1_12", "1_13"}, {"11_22", "12_22"}, {"22_99"}},
		},
	} {
		t.Run(fmt.Sprintf("depth=%v", x.depth), func(t *testing.T) {
			var got collecter
			_, err := g.Neighbours(c(1), x.depth, got.Traverse)
			assert.NoError(t, err)
			assertComponentOrder(t, x.want, got.rules)
		})
	}
}
