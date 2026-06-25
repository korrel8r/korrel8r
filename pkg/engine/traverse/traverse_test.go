// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

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

func TestTraverserTwoPaths(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	e, err := engine.Build().Rules(
		r("ab", "d:a", "d:b", b.Query("d:b", "ab", 1)),
		r("bc", "d:b", "d:c", b.Query("d:c", "bc", 2)),
		r("ac", "d:a", "d:c", b.Query("d:c", "ac", 3)),
		r("ax", "d:a", "d:x", b.Query("d:x", "ax", 4)),
	).Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)

	t.Run("Neighbours", func(t *testing.T) {
		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
		g, err := Neighbors(context.Background(), e, start, 2)
		require.NoError(t, err)
		wantLines := []string{
			"ab(d:a->d:b)",
			"bc(d:b->d:c)",
			"ac(d:a->d:c)",
			"ax(d:a->d:x)",
		}
		wantNodes := []string{
			"d:a[0]",
			"d:b[1]",
			"d:c[2,3]",
			"d:x[4]",
		}
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, wantLines, g.LineStrings())
			assert.ElementsMatch(t, wantNodes, g.NodeStrings(true))
		}
	})

	t.Run("Goals", func(t *testing.T) {
		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
		g, err := Goals(context.Background(), e, start, []korrel8r.Class{b.Class(("d:c"))})
		require.NoError(t, err)
		wantLines := []string{
			"ab(d:a->d:b)",
			"bc(d:b->d:c)",
			"ac(d:a->d:c)",
		}
		wantNodes := []string{
			"d:a[0]",
			"d:b[1]",
			"d:c[2,3]",
		}
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, wantLines, g.LineStrings())
			assert.ElementsMatch(t, wantNodes, g.NodeStrings(true))
		}
	})
}

func TestTraverserCycle(t *testing.T) {
	t.Run("two_node_cycle", func(t *testing.T) {
		b := mock.NewBuilder("d")
		r := b.Rule
		e, err := engine.Build().Rules(
			r("ab", "d:a", "d:b", b.Query("d:b", "ab", 1)),
			r("ba", "d:b", "d:a", b.Query("d:a", "ba", 2)),
		).Stores(b.Store("d", nil)).Engine()
		require.NoError(t, err)

		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
		g, err := Neighbors(context.Background(), e, start, 2)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"ab(d:a->d:b)", "ba(d:b->d:a)"}, g.LineStrings())
		assert.ElementsMatch(t, []string{"d:a[0,2]", "d:b[1]"}, g.NodeStrings(true))
	})

	t.Run("three_node_cycle", func(t *testing.T) {
		b := mock.NewBuilder("d")
		r := b.Rule
		e, err := engine.Build().Rules(
			r("ab", "d:a", "d:b", b.Query("d:b", "ab", 1)),
			r("bc", "d:b", "d:c", b.Query("d:c", "bc", 2)),
			r("ca", "d:c", "d:a", b.Query("d:a", "ca", 3)),
		).Stores(b.Store("d", nil)).Engine()
		require.NoError(t, err)

		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
		g, err := Neighbors(context.Background(), e, start, 3)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"ab(d:a->d:b)", "bc(d:b->d:c)", "ca(d:c->d:a)"}, g.LineStrings())
		assert.ElementsMatch(t, []string{"d:a[0,3]", "d:b[1]", "d:c[2]"}, g.NodeStrings(true))
	})

	t.Run("cycle_with_branch", func(t *testing.T) {
		b := mock.NewBuilder("d")
		r := b.Rule
		e, err := engine.Build().Rules(
			r("ab", "d:a", "d:b", b.Query("d:b", "ab", 1)),
			r("bc", "d:b", "d:c", b.Query("d:c", "bc", 2)),
			r("ca", "d:c", "d:a", b.Query("d:a", "ca", 3)),
			r("ad", "d:a", "d:d", b.Query("d:d", "ad", 4)),
		).Stores(b.Store("d", nil)).Engine()
		require.NoError(t, err)

		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
		g, err := Neighbors(context.Background(), e, start, 3)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			"ab(d:a->d:b)",
			"bc(d:b->d:c)",
			"ca(d:c->d:a)",
			"ad(d:a->d:d)",
		}, g.LineStrings())
		assert.ElementsMatch(t, []string{
			"d:a[0,3]", "d:b[1]", "d:c[2]", "d:d[4]",
		}, g.NodeStrings(true))
	})
}

// TestTraverserCycleUniqueQueries tests that cyclic rules generating unique queries
// on every application (defeating query deduplication) do not cause infinite recursion.
func TestTraverserCycleUniqueQueries(t *testing.T) {
	b := mock.NewBuilder("d")
	var counter atomic.Int64

	// Each application generates a new unique query with a new unique object,
	// so queryBox deduplication cannot stop the cycle.
	uniqueRule := func(goalClass string) mock.ApplyFunc {
		return func(start korrel8r.Object) ([]korrel8r.Query, error) {
			n := counter.Add(1)
			return []korrel8r.Query{
				b.Query(goalClass, fmt.Sprintf("unique-%d", n), fmt.Sprintf("obj-%d", n)),
			}, nil
		}
	}

	e, err := engine.Build().Rules(
		b.Rule("ab", "d:a", "d:b", uniqueRule("d:b")),
		b.Rule("ba", "d:b", "d:a", uniqueRule("d:a")),
	).Stores(b.Store("d", nil)).Engine()
	require.NoError(t, err)

	t.Run("Neighbors", func(t *testing.T) {
		counter.Store(0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{"start"}}
		g, err := Neighbors(ctx, e, start, 5)
		require.NoError(t, err, "should terminate at depth limit without context cancellation")
		assert.NotEmpty(t, g.NodeStrings(true), "should have results")
		t.Logf("nodes: %v", g.NodeStrings(true))
		t.Logf("lines: %v", g.LineStrings())
	})

	t.Run("Goals", func(t *testing.T) {
		counter.Store(0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{"start"}}
		g, err := Goals(ctx, e, start, []korrel8r.Class{b.Class("d:b")})
		require.NoError(t, err, "should terminate without context cancellation")
		assert.NotEmpty(t, g.NodeStrings(true), "should have results")
		t.Logf("nodes: %v", g.NodeStrings(true))
		t.Logf("lines: %v", g.LineStrings())
	})
}

func TestTraverserNeighbors(t *testing.T) {
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
			depth: 0,
			lines: []string{},
			nodes: []string{"d:a[0]"},
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
	} {
		t.Run(fmt.Sprintf("depth %v", x.depth), func(t *testing.T) {
			start := Start{Class: b.Class("d:a"), Objects: []korrel8r.Object{0}}
			g, err := Neighbors(context.Background(), e, start, x.depth)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, g.LineStrings())
				assert.ElementsMatch(t, x.nodes, g.NodeStrings(true))
			}
		})
	}
}
