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
			want: []MultiPath{
				{links("a", "c"), links("c", "z", "cz1", "cz2")},
			},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := New("test", x.rules, nil)
			got, err := g.ShortestPaths(mock.Class("a"), mock.Class("z"))
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, got, "%v != %v", x.want, got)
		})
	}
}

// func rm(start, goal string, extras ...string) korrel8.Rule {
// 	return mock.NewRule(start+"_"+goal, start, goal,
// 		func(o korrel8.Object, _ *korrel8.Constraint) (*korrel8.Query, error) {
// 			for _, s := range extras {
// 				if s == o.(mock.Object).Class().String() {
// 					return nil, nil // Accept
// 				}
// 			}
// 			return nil, fmt.Errorf("no match: %+v in %v", o, extras)
// 		})
// }

// func (s string) korrel8.Class { return mock.Class(s) }

// func TestGraph_Matches(t *testing.T) {
// 	for _, x := range []struct {
// 		name    string
// 		rule    korrel8.Rule
// 		classes []korrel8.Class
// 		want    []korrel8.Class
// 	}{
// 		{
// 			name:    "no match",
// 			rule:    rm("a", "b"),
// 			classes: []korrel8.Class{c("a"), c("b"), c("x"), c("y")},
// 			want:    nil,
// 		},
// 		{
// 			name:    "2 matches",
// 			rule:    rm("a", "b", "x", "y"),
// 			classes: []korrel8.Class{c("a"), c("b"), c("x"), c("y")},
// 			want:    []korrel8.Class{c("x"), c("y")},
// 		},
// 	} {
// 		t.Run(x.name, func(t *testing.T) {
// 			g := New(nil, x.classes)
// 			got := g.extras(x.rule)
// 			assert.ElementsMatch(t, x.want, got)
// 		})
// 	}
// }
// FIXME wildcard
