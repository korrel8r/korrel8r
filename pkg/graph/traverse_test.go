// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

type ruleCollecter struct {
	rules []string
	nodes []int
}

func (c *ruleCollecter) Node(n *Node) {
	c.nodes = append(c.nodes, nodeToInt(n))
}

func (c *ruleCollecter) Line(l *Line) bool {
	c.rules = append(c.rules, l.Rule.Name())
	return true
}

func TestTraverse(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) rule { return rm.r(i, j) }

	for _, x := range []struct {
		name  string
		graph []rule
		rules [][]string // inner slices are are unordered components.
		nodes []int
	}{
		{
			name:  "multipath",
			graph: []rule{r(1, 11), r(1, 12), r(11, 99), r(12, 99)},
			rules: [][]string{{"1_11", "1_12"}, {"11_99", "12_99"}},
			nodes: []int{1, 11, 12, 99},
		},
		{
			name:  "simple",
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 5)},
			rules: [][]string{{"1_2"}, {"2_3"}, {"3_4"}, {"4_5"}},
			nodes: []int{1, 2, 3, 4, 5},
		},
		{
			name:  "cycle", // cycle of 2,3,4
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 2), r(4, 5)},
			rules: [][]string{{"1_2", "2_3", "3_4", "4_5"}},
			nodes: []int{1, 2, 3, 4, 5},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			var got ruleCollecter
			testGraph(x.graph).GoalSearch(x.graph[0].Start()[0], x.graph[len(x.graph)-1].Goal(), &got)
			assertComponentOrder(t, x.rules, got.rules)
			assert.ElementsMatch(t, x.nodes, got.nodes)
		})
	}

}

func TestNeighbours(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	g := testGraph([]rule{r(1, 11), r(11, 1), r(1, 12), r(1, 13), r(11, 22), r(12, 22), r(12, 13), r(22, 99)})
	for _, x := range []struct {
		depth int
		rules [][]string
		nodes []int
	}{
		{
			depth: 0,
			rules: nil,
			nodes: []int{1},
		},
		{
			depth: 1,
			rules: [][]string{{"1_11", "1_12", "1_13"}},
			nodes: []int{1, 11, 12, 13},
		},
		{
			depth: 2,
			rules: [][]string{{"1_11", "1_12", "1_13"}, {"11_22", "12_22"}},
			nodes: []int{1, 11, 12, 13, 22},
		},
		{
			depth: 3,
			rules: [][]string{{"1_11", "1_12", "1_13"}, {"11_22", "12_13", "12_22"}, {"22_99"}},
			nodes: []int{1, 11, 12, 13, 22, 99},
		},
	} {
		t.Run(fmt.Sprintf("depth=%v", x.depth), func(t *testing.T) {
			var got ruleCollecter
			g.Neighbours(c(1), x.depth, &got)
			assert.ElementsMatch(t, x.nodes, got.nodes)
			assertComponentOrder(t, x.rules, got.rules)
		})
	}
}
