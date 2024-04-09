// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestTraverse(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) rule { return rm.r(i, j) }

	for _, x := range []struct {
		name        string
		graph       []rule
		wantRules   [][]rule
		wantClasses []korrel8r.Class
	}{
		{
			name:        "multipath",
			graph:       []rule{r(1, 11), r(1, 12), r(11, 99), r(12, 99)},
			wantRules:   [][]rule{{r(1, 11), r(1, 12)}, {r(11, 99), r(12, 99)}},
			wantClasses: []korrel8r.Class{c(1), c(11), c(12), c(99)},
		},
		{
			name:      "simple",
			graph:     []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 5)},
			wantRules: [][]rule{{r(1, 2)}, {r(2, 3)}, {r(3, 4)}, {r(4, 5)}},
		},
		{
			name:      "cycle", // cycle of 2,3,4
			graph:     []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 2), r(4, 5)},
			wantRules: [][]rule{{r(1, 2)}, {r(2, 3), r(3, 4), r(4, 2)}, {r(4, 5)}},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var (
				gotRules   []rule
				gotClasses []korrel8r.Class
			)
			err := g.Traverse(MakeVisitor(
				func(n *Node, _ Lines) { gotClasses = append(gotClasses, ClassFor(n)) },
				func(l *Line) { gotRules = append(gotRules, RuleFor(l)) },
			))
			assert.NoError(t, err)
			assertComponentOrder(t, x.wantRules, gotRules)
		})
	}
}

func TestNeighbours(t *testing.T) {
	rm := ruleMap{}
	r := func(i, j int) korrel8r.Rule { return rm.r(i, j) }

	g := testGraph([]rule{r(1, 11), r(1, 12), r(1, 13), r(11, 22), r(12, 22), r(12, 13), r(22, 99)})
	for _, x := range []struct {
		depth int
		want  [][]rule
	}{
		{
			depth: 1,
			want:  [][]rule{{r(1, 11), r(1, 12), r(1, 13)}},
		},
		{
			depth: 2,
			want:  [][]rule{{r(1, 11), r(1, 12), r(1, 13)}, {r(11, 22), r(12, 22)}},
		},
		{
			depth: 3,
			want:  [][]rule{{r(1, 11), r(1, 12), r(1, 13)}, {r(11, 22), r(12, 22)}, {r(22, 99)}},
		},
	} {
		t.Run(fmt.Sprintf("depth=%v", x.depth), func(t *testing.T) {
			var (
				gotRules []rule
			)
			g.Neighbours(c(1), x.depth, MakeVisitor(
				func(n *Node, _ Lines) {},
				func(l *Line) { gotRules = append(gotRules, RuleFor(l)) },
			))
			assertComponentOrder(t, x.want, gotRules)
		})
	}
}
