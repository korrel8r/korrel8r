// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraverse(t *testing.T) {
	for _, x := range []struct {
		name  string
		graph []rule
		want  [][]rule
	}{
		{
			name:  "multipath",
			graph: []rule{r(1, 11), r(1, 12), r(11, 99), r(12, 99)},
			want:  [][]rule{{r(1, 11), r(1, 12)}, {r(11, 99), r(12, 99)}},
		},
		{
			name:  "simple",
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 5)},
			want:  [][]rule{{r(1, 2)}, {r(2, 3)}, {r(3, 4)}, {r(4, 5)}},
		},
		{
			name:  "cycle", // cycle of 2,3,4
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 2), r(4, 5)},
			want:  [][]rule{{r(1, 2)}, {r(2, 3), r(3, 4), r(4, 2)}, {r(4, 5)}},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var got []rule
			err := g.Traverse(func(l *Line) { got = append(got, RuleFor(l)) })
			assert.NoError(t, err)
			assertComponentOrder(t, x.want, got)
		})
	}
}

func TestNeighbours(t *testing.T) {
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
			want:  [][]rule{{r(1, 11), r(1, 12), r(1, 13)}, {r(11, 22), r(12, 22), r(12, 13)}},
		},
		{
			depth: 3,
			want:  [][]rule{{r(1, 11), r(1, 12), r(1, 13)}, {r(11, 22), r(12, 22), r(12, 13)}, {r(22, 99)}},
		},
	} {
		t.Run(fmt.Sprintf("depth=%v", x.depth), func(t *testing.T) {
			var got []rule
			g.Neighbours(c(1), x.depth, func(l *Line) { got = append(got, RuleFor(l)) })
			assertComponentOrder(t, x.want, got)
		})
	}
}
