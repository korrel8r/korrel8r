package korrel8

import (
	"fmt"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

// Dummy rule and class

type class string

func (c class) Domain() Domain { return "fake" }

type rule struct {
	start, goal Class
	query       Result
}

func (r rule) Start() Class                  { return r.start }
func (r rule) Goal() Class                   { return r.goal }
func (r rule) String() string                { return fmt.Sprintf("(%v)->%v", r.start, r.goal) }
func (r rule) Follow(Object) (Result, error) { return r.query, nil }

func tr(start, goal string) rule { return rule{start: class(start), goal: class(goal)} }

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
