package graph

import (
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
)

var r = mock.QuickRule

func nr(name, start, goal string) korrel8.Rule { return mock.NewRule(name, start, goal, nil) }

func l(startGoal ...string) Links { return append(Links{}, mock.Rules(startGoal...)...) }

func AssertMultiPathEqual(t *testing.T, a, b MultiPath) bool {
	if !assert.Equal(t, len(a), len(b), "lengths not equal") {
		return false
	}
	for i := range a {
		if !assert.ElementsMatch(t, a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestGraph_ShortestPaths(t *testing.T) {
	for _, x := range []struct {
		name  string
		rules []korrel8.Rule
		want  []MultiPath
	}{
		{
			name:  "simple",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("c", "z")},
			want:  []MultiPath{{l("a", "b"), l("b", "c"), l("c", "z")}},
		},
		{
			name:  "shortest",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("c", "d"), r("d", "z"), r("c", "z"), r("a", "z")},
			want:  []MultiPath{{l("a", "z")}},
		},
		{
			name:  "none",
			rules: []korrel8.Rule{r("a", "b"), r("c", "z")},
			want:  nil,
		},
		{
			name:  "multi-pick-shortest",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("a", "c"), r("c", "z")},
			want:  []MultiPath{{links("a", "c"), links("c", "z")}},
		},
		{
			name:  "multi-shortest",
			rules: []korrel8.Rule{r("a", "b"), r("b", "c"), r("b", "y"), r("y", "z"), r("c", "z")},
			want: []MultiPath{
				{links("a", "b"), links("b", "c"), links("c", "z")},
				{links("a", "b"), links("b", "y"), links("y", "z")},
			},
		},
		{
			name:  "multi-link",
			rules: []korrel8.Rule{r("a", "c"), nr("cz1", "c", "z"), nr("cz2", "c", "z")},
			want:  []MultiPath{{links("a", "c"), links("c", "z", "cz1", "cz2")}},
		},
		{
			name:  "multi-link-and-path",
			rules: []korrel8.Rule{r("a", "c"), nr("cz1", "c", "z"), nr("cz2", "c", "z")},
			want:  []MultiPath{{links("a", "c"), links("c", "z", "cz1", "cz2")}}},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := New("test", x.rules, nil)
			got, err := g.ShortestPaths(mock.Class("a"), mock.Class("z"))
			assert.NoError(t, err)
			if assert.Equal(t, len(x.want), len(got)) {
				for i := range got {
					x.want[i].Sort()
					got[i].Sort()
					assert.ElementsMatch(t, x.want, got)
				}
			}
		})
	}
}
