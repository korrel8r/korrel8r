package korrel8

import (
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
