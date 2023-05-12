// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_allPaths(t *testing.T) {
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
			paths := g.AllPaths(class(1), class(13))
			assert.Equal(t, x.want, graphRules(paths))
		})
	}
}
