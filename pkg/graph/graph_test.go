// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestGoalPaths(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	// Paths:
	// a-(b1,b2)-c len(3)
	// a-(b1,b2)-c-d len(4)
	// a-x-y-z-d len(5)
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("bc1", "d:b", "d:c", nil),
		r("bc2", "d:b", "d:c", nil),
		r("cd", "d:c", "d:d", nil),
		r("ax", "d:a", "d:x", nil),
		r("xy", "d:x", "d:y", nil),
		r("yz", "d:y", "d:z", nil),
		r("za", "d:z", "d:a", nil), // Cycle
		r("zd", "d:z", "d:d", nil),
		r("ze", "d:z", "d:e", nil),
		r("ed", "d:e", "d:d", nil),
	).FullGraph()

	for _, x := range []struct {
		name         string
		start        korrel8r.Class
		goals        []korrel8r.Class
		lines, nodes []string
	}{
		{
			name:  "shortest",
			start: b.Class("d:a"),
			goals: b.Classes("d:c"),
			lines: []string{"ab(d:a->d:b)", "bc1(d:b->d:c)", "bc2(d:b->d:c)"},
			nodes: []string{"d:a", "d:b", "d:c"},
		},
		{
			name:  "k-shortest+1",
			start: b.Class("d:a"),
			goals: b.Classes("d:d"),
			lines: []string{
				"ab(d:a->d:b)", "bc1(d:b->d:c)", "bc2(d:b->d:c)", "cd(d:c->d:d)",
				"ax(d:a->d:x)", "xy(d:x->d:y)", "yz(d:y->d:z)", "zd(d:z->d:d)"},
			nodes: []string{"d:a", "d:b", "d:c", "d:d", "d:x", "d:y", "d:z"},
		},
		{
			name:  "weighted shortest",
			start: b.Class("d:a"),
			goals: b.Classes("d:d"),
			lines: []string{
				"ab(d:a->d:b)", "bc1(d:b->d:c)", "bc2(d:b->d:c)", "cd(d:c->d:d)",
				"ax(d:a->d:x)", "xy(d:x->d:y)", "yz(d:y->d:z)", "zd(d:z->d:d)"},
			nodes: []string{"d:a", "d:b", "d:c", "d:d", "d:x", "d:y", "d:z"},
		},
		{
			name:  "cycle",
			start: b.Class("d:a"),
			goals: b.Classes("d:z"),
			lines: []string{"ax(d:a->d:x)", "xy(d:x->d:y)", "yz(d:y->d:z)"},
			nodes: []string{"d:a", "d:x", "d:y", "d:z"},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			sub, err := g.GoalPaths(x.start, x.goals)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, sub.LineStrings(), "%#v", sub.LineStrings())
				assert.ElementsMatch(t, x.nodes, sub.NodeStrings(true), "%#v", sub.NodeStrings(true))
			}
		})
	}
}

func TestNeighbours(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("ac", "d:a", "d:c", nil),
		r("bx", "d:b", "d:x", nil),
		r("cy", "d:c", "d:y", nil),
		r("yz", "d:y", "d:z", nil),
		r("zq", "d:z", "d:q", nil),
	).FullGraph()

	for _, x := range []struct {
		depth int
		lines []string
		nodes []string
	}{
		{
			depth: 4,
			lines: []string{
				"ab(d:a->d:b)",
				"ac(d:a->d:c)",
				"bx(d:b->d:x)",
				"cy(d:c->d:y)",
				"yz(d:y->d:z)",
				"zq(d:z->d:q)",
			},
			nodes: []string{
				"d:a",
				"d:b",
				"d:c",
				"d:x",
				"d:y",
				"d:z",
				"d:q",
			},
		},
		{
			depth: 3,
			lines: []string{
				"ab(d:a->d:b)",
				"ac(d:a->d:c)",
				"bx(d:b->d:x)",
				"cy(d:c->d:y)",
				"yz(d:y->d:z)",
			},
			nodes: []string{
				"d:a",
				"d:b",
				"d:c",
				"d:x",
				"d:y",
				"d:z",
			},
		},
		{
			depth: 2,
			lines: []string{
				"ab(d:a->d:b)",
				"ac(d:a->d:c)",
				"bx(d:b->d:x)",
				"cy(d:c->d:y)",
			},
			nodes: []string{
				"d:a",
				"d:b",
				"d:c",
				"d:x",
				"d:y",
			},
		},
		{
			depth: 1, lines: []string{
				"ab(d:a->d:b)",
				"ac(d:a->d:c)",
			},
			nodes: []string{
				"d:a",
				"d:b",
				"d:c",
			},
		},
		{
			depth: 0,
			lines: []string{},
			nodes: []string{"d:a"},
		},
	} {
		t.Run(fmt.Sprintf("depth %v", x.depth), func(t *testing.T) {
			sub, err := g.Neighbours(b.Class("d:a"), x.depth)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, sub.LineStrings())
				assert.ElementsMatch(t, x.nodes, sub.NodeStrings(true))
			}
		})
	}
}
