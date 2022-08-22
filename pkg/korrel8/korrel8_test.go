package korrel8

import (
	"fmt"
	"sort"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

// Dummy rule and class

type class string

func (c class) String() string    { return string(c) }
func (c class) Contains(any) bool { return false }

type rule struct{ context, goal Class }

func (r rule) Context() Class   { return r.context }
func (r rule) Goal() Class      { return r.goal }
func (r rule) String() string   { return fmt.Sprintf("(%v)->%v", r.context, r.goal) }
func (r rule) Follow(any) Query { return "" }

func tr(context, goal string) rule { return rule{context: class(context), goal: class(goal)} }

func sortPaths(p [][]Rule) [][]Rule {
	sort.Slice(p, func(i, j int) bool { return fmt.Sprintf("%v", p[i]) < fmt.Sprintf("%v", p[j]) })
	return p
}

func TestFindPaths(t *testing.T) {
	for i, g := range []struct {
		rules *Correlator
		want  [][]Rule
	}{
		{
			rules: New(tr("a", "b"), tr("a", "c"), tr("c", "x"), tr("c", "y"), tr("y", "z")),
			want: [][]Rule{
				{tr("a", "c"), tr("c", "y"), tr("y", "z")},
			},
		},
		{
			rules: New(tr("a", "b"), tr("a", "c"), tr("c", "x"), tr("b", "y"), tr("c", "y"), tr("y", "z"), tr("z", "zz")),
			want: [][]Rule{
				{tr("a", "b"), tr("b", "y"), tr("y", "z")},
				{tr("a", "c"), tr("c", "y"), tr("y", "z")},
			},
		},
		{
			rules: New(tr("a", "b"), tr("a", "c"), tr("a", "z"), tr("b", "y"), tr("c", "y"), tr("y", "z")),
			want: [][]Rule{
				{tr("a", "b"), tr("b", "y"), tr("y", "z")},
				{tr("a", "c"), tr("c", "y"), tr("y", "z")},
				{tr("a", "z")},
			},
		},
		{
			rules: New(tr("a", "b"), tr("a", "c"), tr("c", "x"), tr("y", "z")),
			want:  [][]Rule{},
		}} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			rc := g.rules.findPaths(class("a"), class("z"))
			assert.Equal(t, fmt.Sprintf("%v", sortPaths(g.want)), fmt.Sprintf("%v", sortPaths(rc)))
		})
	}
}
