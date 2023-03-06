package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestTraverse(t *testing.T) {
	for _, x := range []struct {
		name   string
		graph  []rule
		want   [][]rule // All possible orderings.
		sorted []int
		cycles [][]int
	}{
		{
			name:  "simple",
			graph: []rule{r(1, 2), r(2, 3), r(1, 3), r(3, 4), r(4, 13)},
			want: [][]rule{
				{r(1, 2), r(2, 3), r(1, 3), r(3, 4), r(4, 13)},
				{r(1, 2), r(1, 3), r(2, 3), r(3, 4), r(4, 13)}},
		},
		{
			name:  "inner-cycle",
			graph: []rule{r(1, 2), r(2, 3), r(3, 4), r(4, 3), r(4, 13)},
			want:  [][]rule{{r(1, 2), r(2, 3), r(3, 4), r(4, 13)}},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var got []rule
			// Use the stable sort for equality testing. Equivalent to the faster unstable sort.
			err := g.Traverse(func(e Edge) {
				e.Reset()
				for e.Next() {
					got = append(got, e.Line().Rule.(rule))
				}
				slices.SortFunc(got, ruleLess)
			})

			assert.NoError(t, err)
			assert.Contains(t, x.want, got)
		})
	}
}
