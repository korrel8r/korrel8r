package korrel8

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

// Dummy implementations

type class string

func (c class) Domain() Domain { return "fake" }

type object string

func (o object) Class() Class           { return class("thing") }
func (o object) Identifier() Identifier { return Identifier(o) }

type rule struct {
	start, goal Class
	result      Result
}

func (r rule) Start() Class                  { return r.start }
func (r rule) Goal() Class                   { return r.goal }
func (r rule) String() string                { return fmt.Sprintf("(%v)->%v", r.start, r.goal) }
func (r rule) Follow(Object) (Result, error) { return r.result, nil }

func tr(start, goal string) rule { return rule{start: class(start), goal: class(goal)} }

type store struct{}

func (s store) Query(_ context.Context, q string) ([]Object, error) {
	var objs []Object
	for _, s := range strings.Split(q, ",") {
		objs = append(objs, object(s))
	}
	return objs, nil
}

func TestRules_Path(t *testing.T) {
	for i, g := range []struct {
		rules *Rules
		want  []Path
	}{
		{
			rules: NewRules(tr("a", "b"), tr("a", "c"), tr("c", "x"), tr("c", "y"), tr("y", "z")),
			want: []Path{
				{tr("a", "c"), tr("c", "y"), tr("y", "z")},
			},
		},
		{
			rules: NewRules(tr("a", "b"), tr("a", "c"), tr("c", "x"), tr("b", "y"), tr("c", "y"), tr("y", "z"), tr("z", "zz")),
			want: []Path{
				{tr("a", "b"), tr("b", "y"), tr("y", "z")},
				{tr("a", "c"), tr("c", "y"), tr("y", "z")},
			},
		},
		{
			rules: NewRules(tr("a", "b"), tr("a", "c"), tr("a", "z"), tr("b", "y"), tr("c", "y"), tr("y", "z")),
			want: []Path{
				{tr("a", "b"), tr("b", "y"), tr("y", "z")},
				{tr("a", "c"), tr("c", "y"), tr("y", "z")},
				{tr("a", "z")},
			},
		},
		{
			rules: NewRules(tr("a", "b"), tr("a", "c"), tr("c", "x"), tr("y", "z")),
			want:  []Path{},
		}} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := g.rules.Paths(class("a"), class("z"))
			assert.ElementsMatch(t, g.want, got)
		})
	}
}

func TestResult_Get(t *testing.T) {
	for _, x := range []struct {
		r    Result
		want []object
	}{
		{
			r:    Result{Domain: "fake", Queries: []string{"a,b,c", "b,c,d", "x,y,x"}},
			want: []object{"a", "b", "c", "d", "x", "y"},
		},
		{
			r:    Result{Domain: "fake", Queries: nil},
			want: nil,
		},
	} {
		objs, _ := x.r.Get(context.Background(), store{})
		assert.ElementsMatch(t, x.want, objs)
	}
}
