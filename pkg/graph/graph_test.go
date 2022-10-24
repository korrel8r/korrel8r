package graph

import (
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
)

func r(start, goal string) korrel8.Rule { return mock.NewRule(start, goal, nil) }

func TestGraph_ShortestPath(t *testing.T) {
	for _, x := range []struct {
		name        string
		rules, want []korrel8.Rule
	}{
		{
			name:  "simple",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("c", "z")},
			want:  []korrel8.Rule{r("a", "b"), r("b", "c"), r("c", "z")},
		},
		{
			name:  "shortest",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("c", "d"), r("d", "z"), r("c", "z"), r("a", "z")},
			want:  []korrel8.Rule{r("a", "z")},
		},
		{
			name:  "none",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("c", "d")},
			want:  nil,
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := New()
			g.Add(x.rules)
			got := g.ShortestPath(mock.Class("a"), mock.Class("z"))
			assert.ElementsMatch(t, x.want, got, g.graph.String())
		})
	}
}
