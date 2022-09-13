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
			objs, _ := Result(x.queries).Get(context.Background(), mockStore{})
			assert.ElementsMatch(t, x.want, objs)
		})
	}
}

func TestPath_Follow(t *testing.T) {
	for i, x := range []struct {
		path Path
		want Result
	}{
		{
			path: Path{
				// Return 2 results, must follow both
				rr("a", "b", func(Object) Result { return []string{"1.b", "2.b"} }),
				// Return the start object's name with the goal class.
				rr("b", "c", func(start Object) Result { return []string{start.(mockObject).name + ".c", "x.c"} }),
				rr("c", "z", func(start Object) Result { return []string{start.(mockObject).name + ".z", "y.z"} }),
			},
			want: Result{"1.z", "2.z", "x.z", "y.z"},
		},
		{
			path: nil,
			want: Result{},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := x.path.Follow(context.Background(), o("foo", "a"), mockStores)
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, r)
		})
	}
}

func TestRule_FollowEach(t *testing.T) {
	for i, x := range []struct {
		rule  Rule
		start []Object
		want  Result
	}{
		{
			rule:  rr("a", "b", func(start Object) Result { return []string{start.(mockObject).name + ".b", "x.b"} }),
			start: []Object{o("1", "a"), o("2", ".a"), o("1", "a")},
			want:  Result{"1.b", "2.b", "x.b"},
		},
		{
			rule: r("a", "b"),
			want: Result{},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := FollowEach(x.rule, x.start)
			assert.NoError(t, err)
			assert.ElementsMatch(t, x.want, r)
		})
	}
}
