package graph

import (
	"fmt"
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
			g := New(x.rules, nil)
			got := g.ShortestPath(mock.Class("a"), mock.Class("z"))
			assert.ElementsMatch(t, x.want, got)
		})
	}
}

func rm(start, goal string, extras ...string) korrel8.Rule {
	return mock.NewRule(start, goal,
		func(o korrel8.Object, _ *korrel8.Constraint) (*korrel8.Query, error) {
			for _, s := range extras {
				if s == o.(mock.Object).Class.String() {
					return nil, nil // Accept
				}
			}
			return nil, fmt.Errorf("no match: %+v in %v", o, extras)
		})
}
func c(s string) korrel8.Class { return mock.Class(s) }

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
