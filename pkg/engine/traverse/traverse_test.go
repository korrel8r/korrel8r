// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraverserGoals(t *testing.T) {
	b := mock.NewBuilder("d")
	e, err := engine.Build().Rules(
		// Return 2 results, must follow both
		b.Rule("ab", "d:a", "d:b", b.Query("d:b", "", 1, 2)),
		// 2 rules, must follow both. Incorporate data from start object.
		b.Rule("bc1", "d:b", "d:c", func(start korrel8r.Object) ([]korrel8r.Query, error) {
			return []korrel8r.Query{b.Query("d:c", fmt.Sprintf("bc1/%v", start), start)}, nil
		}),
		b.Rule("bc2", "d:b", "d:c", func(start korrel8r.Object) ([]korrel8r.Query, error) {
			result := start.(int) + 10
			return []korrel8r.Query{b.Query("d:c", fmt.Sprintf("bc2/%v", start), result)}, nil
		}),
		b.Rule("cz", "d:c", "d:z", func(start korrel8r.Object) ([]korrel8r.Query, error) {
			return []korrel8r.Query{b.Query("d:z", fmt.Sprintf("cz/%v", start), start)}, nil
		}),
	).Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)

	for _, x := range []struct {
		start        Start
		goals        []korrel8r.Class
		lines, nodes []string
	}{
		{
			start: Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}},
			goals: []korrel8r.Class{b.Class("d:z")},
			lines: []string{
				"ab(d:a->d:b)",
				"bc1(d:b->d:c)",
				"bc2(d:b->d:c)",
				"cz(d:c->d:z)",
			},
			nodes: []string{
				"d:a[0]",
				"d:b[1,2]",
				"d:c[1,11,12,2]",
				"d:z[1,11,12,2]",
			}},
	} {
		t.Run(fmt.Sprintf("%v->%v", x.start.Class, x.goals), func(t *testing.T) {
			g, err := Goals(context.Background(), e, x.start, x.goals)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, g.LineStrings())
				assert.ElementsMatch(t, x.nodes, g.NodeStrings(true))
			}
		})
	}
}

func TestTraverserNeighbours(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	e, err := engine.Build().Rules(
		r("ab", "d:a", "d:b", b.Query("d:b", "ab", 1)),
		r("ac", "d:a", "d:c", b.Query("d:c", "ac", 2)),
		r("bx", "d:b", "d:x", b.Query("d:x", "bx", 3)),
		r("cy", "d:c", "d:y", b.Query("d:y", "cy", 4)),
		r("yz", "d:y", "d:z", b.Query("d:z", "yz", 5)),
		r("zq", "d:z", "d:q", b.Query("d:q", "zq", 6)),
	).Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)

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
				"d:a[0]",
				"d:b[1]",
				"d:c[2]",
				"d:x[3]",
				"d:y[4]",
				"d:z[5]",
				"d:q[6]",
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
				"d:a[0]",
				"d:b[1]",
				"d:c[2]",
				"d:x[3]",
				"d:y[4]",
				"d:z[5]",
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
				"d:a[0]",
				"d:b[1]",
				"d:c[2]",
				"d:x[3]",
				"d:y[4]",
			},
		},
		{
			depth: 1, lines: []string{
				"ab(d:a->d:b)",
				"ac(d:a->d:c)",
			},
			nodes: []string{
				"d:a[0]",
				"d:b[1]",
				"d:c[2]",
			},
		},
		{
			depth: 0,
			lines: []string{},
			nodes: []string{
				"d:a[0]",
			},
		},
	} {
		t.Run(fmt.Sprintf("depth %v", x.depth), func(t *testing.T) {
			start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
			g, err := Neighbours(context.Background(), e, start, x.depth)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, g.LineStrings())
				assert.ElementsMatch(t, x.nodes, g.NodeStrings(true))
			}
		})
	}
}
