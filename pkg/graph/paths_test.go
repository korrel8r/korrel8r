package graph

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestAllPaths(t *testing.T) {
	for _, x := range []struct {
		name  string
		graph []korrel8r.Rule
		want  [][]int
	}{
		{
			name:  "simple",
			graph: []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(3, 12), r(12, 13)},
			want:  [][]int{{1, 3, 12, 13}},
		},
		{
			name:  "multiple",
			graph: []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(2, 12), r(3, 12), r(12, 13)},
			want:  [][]int{{1, 2, 12, 13}, {1, 3, 12, 13}},
		},
		{
			name:  "lengths",
			graph: []korrel8r.Rule{r(1, 2), r(1, 3), r(1, 13), r(2, 12), r(3, 12), r(12, 13)},
			want:  [][]int{{1, 2, 12, 13}, {1, 3, 12, 13}, {1, 13}},
		},
		{
			name:  "empty",
			graph: []korrel8r.Rule{r(1, 2), r(1, 3), r(3, 11), r(12, 13)},
			want:  nil,
		}} {
		t.Run(x.name, func(t *testing.T) {
			g := New(x.graph...)
			paths := AllPaths(g, g.ClassID(class(1)), g.ClassID(class(13)))

			assert.Equal(t, x.want, pathsToInts(paths))
		})
	}
}
