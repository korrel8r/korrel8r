package korrel8

import (
	"context"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

func TestRuleSet_FindPath(t *testing.T) {
	for i, g := range []struct {
		rules *RuleSet
		want  []Path
	}{
		{
			rules: NewRuleSet(r("a", "b"), r("a", "c"), r("c", "x"), r("c", "y"), r("y", "z")),
			want: []Path{
				{r("a", "c"), r("c", "y"), r("y", "z")},
			},
		},
		{
			rules: NewRuleSet(r("a", "b"), r("a", "c"), r("c", "x"), r("b", "y"), r("c", "y"), r("y", "z"), r("z", "zz")),
			want: []Path{
				{r("a", "b"), r("b", "y"), r("y", "z")},
				{r("a", "c"), r("c", "y"), r("y", "z")},
			},
		},
		{
			rules: NewRuleSet(r("a", "b"), r("a", "c"), r("a", "z"), r("b", "y"), r("c", "y"), r("y", "z")),
			want: []Path{
				{r("a", "b"), r("b", "y"), r("y", "z")},
				{r("a", "c"), r("c", "y"), r("y", "z")},
				{r("a", "z")},
			},
		},
		{
			rules: NewRuleSet(r("a", "b"), r("a", "c"), r("c", "x"), r("y", "z")),
			want:  []Path{},
		}} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := g.rules.FindPaths(mockClass("a"), mockClass("z"))
			assert.ElementsMatch(t, g.want, got)
		})
	}
}

func TestResult_Get(t *testing.T) {
	for i, x := range []struct {
		queries []string
		want    []mockObject
	}{
		{
			queries: []string{"a.x,b.x,c.x", "b.x,c.y", "e.x,e.y,e.x"},
			want:    []mockObject{{"a", "x"}, {"b", "x"}, {"c", "x"}, {"c", "y"}, {"e", "x"}, {"e", "y"}},
		},
		{
			queries: nil,
			want:    nil,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			objs, _ := Queries(x.queries).Get(context.Background(), mockStore{})
			assert.ElementsMatch(t, x.want, objs)
		})
	}
}
