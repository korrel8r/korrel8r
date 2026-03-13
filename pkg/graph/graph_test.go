// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNeighbors(t *testing.T) {
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
			sub, err := g.Neighbors(b.Class("d:a"), x.depth)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.lines, sub.LineStrings())
				assert.ElementsMatch(t, x.nodes, sub.NodeStrings(true))
			}
		})
	}
}

func TestWeight(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	// Rule with single goal class vs rule with multiple goal classes (more expensive)
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("ab2", "d:a", []string{"d:b", "d:c"}, nil), // multi-goal rule
		r("cd", "d:c", "d:d", nil),
	).FullGraph()

	na := g.NodeFor(b.Class("d:a"))
	nb := g.NodeFor(b.Class("d:b"))
	nc := g.NodeFor(b.Class("d:c"))
	nd := g.NodeFor(b.Class("d:d"))

	// Self-weight is 0
	w, ok := g.Weight(na.ID(), na.ID())
	assert.True(t, ok)
	assert.Equal(t, 0.0, w)

	// a->b has two rules: ab (1 goal) and ab2 (2 goals), min weight = 1
	w, ok = g.Weight(na.ID(), nb.ID())
	assert.True(t, ok)
	assert.Equal(t, 1.0, w)

	// c->d has one rule: cd (1 goal), weight = 1
	w, ok = g.Weight(nc.ID(), nd.ID())
	assert.True(t, ok)
	assert.Equal(t, 1.0, w)

	// No edge from b->a
	w, ok = g.Weight(nb.ID(), na.ID())
	assert.False(t, ok)
	assert.True(t, math.IsInf(w, 1))
}

func TestNodeFor(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(r("ab", "d:a", "d:b", nil)).FullGraph()

	// Existing class
	n := g.NodeFor(b.Class("d:a"))
	assert.NotNil(t, n)
	assert.Equal(t, b.Class("d:a"), n.Class)

	// Non-existent class
	n = g.NodeFor(b.Class("d:z"))
	assert.Nil(t, n)
}

func TestNodeForErr(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(r("ab", "d:a", "d:b", nil)).FullGraph()

	n, err := g.NodeForErr(b.Class("d:a"))
	require.NoError(t, err)
	assert.Equal(t, b.Class("d:a"), n.Class)

	_, err = g.NodeForErr(b.Class("d:z"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "class not found")
}

func TestEachNode(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(r("ab", "d:a", "d:b", nil), r("bc", "d:b", "d:c", nil)).FullGraph()

	var classes []string
	g.EachNode(func(n *Node) { classes = append(classes, n.Class.String()) })
	sort.Strings(classes)
	assert.Equal(t, []string{"d:a", "d:b", "d:c"}, classes)
}

func TestEachEdge(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(r("ab", "d:a", "d:b", nil), r("bc", "d:b", "d:c", nil)).FullGraph()

	var edges []string
	g.EachEdge(func(e *Edge) {
		edges = append(edges, fmt.Sprintf("%v->%v", e.Start().Class, e.Goal().Class))
	})
	sort.Strings(edges)
	assert.Equal(t, []string{"d:a->d:b", "d:b->d:c"}, edges)
}

func TestEachLine(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab1", "d:a", "d:b", nil),
		r("ab2", "d:a", "d:b", nil),
		r("bc", "d:b", "d:c", nil),
	).FullGraph()

	var lines []string
	g.EachLine(func(l *Line) { lines = append(lines, l.Rule.Name()) })
	sort.Strings(lines)
	assert.Equal(t, []string{"ab1", "ab2", "bc"}, lines)
}

func TestEachLineBetween(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab1", "d:a", "d:b", nil),
		r("ab2", "d:a", "d:b", nil),
		r("bc", "d:b", "d:c", nil),
	).FullGraph()

	na := g.NodeFor(b.Class("d:a"))
	nb := g.NodeFor(b.Class("d:b"))

	var lines []string
	g.EachLineBetween(na, nb, func(l *Line) { lines = append(lines, l.Rule.Name()) })
	sort.Strings(lines)
	assert.Equal(t, []string{"ab1", "ab2"}, lines)
}

func TestEachLineFrom(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("ac", "d:a", "d:c", nil),
		r("bd", "d:b", "d:d", nil),
	).FullGraph()

	na := g.NodeFor(b.Class("d:a"))
	var lines []string
	g.EachLineFrom(na, func(l *Line) { lines = append(lines, l.Rule.Name()) })
	sort.Strings(lines)
	assert.Equal(t, []string{"ab", "ac"}, lines)
}

func TestEachLineTo(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("cb", "d:c", "d:b", nil),
		r("bd", "d:b", "d:d", nil),
	).FullGraph()

	nb := g.NodeFor(b.Class("d:b"))
	var lines []string
	g.EachLineTo(nb, func(l *Line) { lines = append(lines, l.Rule.Name()) })
	sort.Strings(lines)
	assert.Equal(t, []string{"ab", "cb"}, lines)
}

func TestSelect(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("bc", "d:b", "d:c", nil),
		r("cd", "d:c", "d:d", nil),
	).FullGraph()

	// Select only lines starting from "d:a"
	sub := g.Select(func(l *Line) bool {
		return l.Start().Class == b.Class("d:a")
	})
	assert.Equal(t, []string{"ab(d:a->d:b)"}, sub.LineStrings())
}

func TestFindLine(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	rule1 := r("ab1", "d:a", "d:b", nil)
	rule2 := r("ab2", "d:a", "d:b", nil)
	g := NewData(rule1, rule2).FullGraph()

	// Find existing rule
	l := g.FindLine(b.Class("d:a"), b.Class("d:b"), rule1)
	require.NotNil(t, l)
	assert.Equal(t, rule1, l.Rule)

	l = g.FindLine(b.Class("d:a"), b.Class("d:b"), rule2)
	require.NotNil(t, l)
	assert.Equal(t, rule2, l.Rule)

	// Non-existent start
	l = g.FindLine(b.Class("d:z"), b.Class("d:b"), rule1)
	assert.Nil(t, l)

	// Non-existent goal
	l = g.FindLine(b.Class("d:a"), b.Class("d:z"), rule1)
	assert.Nil(t, l)

	// Wrong rule for this edge
	rule3 := r("cd", "d:c", "d:d", nil)
	l = g.FindLine(b.Class("d:a"), b.Class("d:b"), rule3)
	assert.Nil(t, l)
}

func TestRemoveEmpty(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(
		r("ab", "d:a", "d:b", nil),
		r("bc", "d:b", "d:c", nil),
	).FullGraph()

	// All nodes and lines are empty initially, RemoveEmpty should remove everything
	g.RemoveEmpty()
	var nodes []string
	g.EachNode(func(n *Node) { nodes = append(nodes, n.Class.String()) })
	assert.Empty(t, nodes)
	assert.Empty(t, g.LineStrings())
}

func TestRemoveEmptyGoalPaths(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	// Helper to create a fresh graph: a->b->c->d, a->x->y
	makeGraph := func() *Graph {
		return NewData(
			r("ab", "d:a", "d:b", nil),
			r("bc", "d:b", "d:c", nil),
			r("cd", "d:c", "d:d", nil),
			r("ax", "d:a", "d:x", nil),
			r("xy", "d:x", "d:y", nil),
		).FullGraph()
	}

	t.Run("keeps path to non-empty goal", func(t *testing.T) {
		g := makeGraph()
		// Make goal node d non-empty
		g.NodeFor(b.Class("d:d")).Result.Append("obj1")

		g.RemoveEmptyGoalPaths(b.Classes("d:d"))

		// Nodes on the path a->b->c->d should remain
		assert.NotNil(t, g.NodeFor(b.Class("d:a")))
		assert.NotNil(t, g.NodeFor(b.Class("d:b")))
		assert.NotNil(t, g.NodeFor(b.Class("d:c")))
		assert.NotNil(t, g.NodeFor(b.Class("d:d")))
		// Nodes x, y are not connected to d, should be removed
		assert.Nil(t, g.NodeFor(b.Class("d:x")))
		assert.Nil(t, g.NodeFor(b.Class("d:y")))
	})

	t.Run("removes all when goal is empty", func(t *testing.T) {
		g := makeGraph()
		// Goal d exists but is empty (no results)
		g.RemoveEmptyGoalPaths(b.Classes("d:d"))

		// All nodes should be removed since goal is empty
		var nodes []string
		g.EachNode(func(n *Node) { nodes = append(nodes, n.Class.String()) })
		assert.Empty(t, nodes)
	})

	t.Run("multiple goals", func(t *testing.T) {
		g := makeGraph()
		// Make both d and y non-empty
		g.NodeFor(b.Class("d:d")).Result.Append("obj1")
		g.NodeFor(b.Class("d:y")).Result.Append("obj2")

		g.RemoveEmptyGoalPaths(b.Classes("d:d", "d:y"))

		// All nodes should remain since every node can reach either d or y
		assert.NotNil(t, g.NodeFor(b.Class("d:a")))
		assert.NotNil(t, g.NodeFor(b.Class("d:b")))
		assert.NotNil(t, g.NodeFor(b.Class("d:c")))
		assert.NotNil(t, g.NodeFor(b.Class("d:d")))
		assert.NotNil(t, g.NodeFor(b.Class("d:x")))
		assert.NotNil(t, g.NodeFor(b.Class("d:y")))
	})

	t.Run("nonexistent goal", func(t *testing.T) {
		g := makeGraph()
		g.RemoveEmptyGoalPaths(b.Classes("d:nonexistent"))

		// All nodes should be removed
		var nodes []string
		g.EachNode(func(n *Node) { nodes = append(nodes, n.Class.String()) })
		assert.Empty(t, nodes)
	})
}

func TestGoalPaths_Errors(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(r("ab", "d:a", "d:b", nil)).FullGraph()

	// Non-existent start
	_, err := g.GoalPaths(b.Class("d:z"), b.Classes("d:b"))
	assert.Error(t, err)

	// Non-existent goal
	_, err = g.GoalPaths(b.Class("d:a"), b.Classes("d:z"))
	assert.Error(t, err)
}

func TestNeighbors_Error(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	g := NewData(r("ab", "d:a", "d:b", nil)).FullGraph()

	_, err := g.Neighbors(b.Class("d:z"), 1)
	assert.Error(t, err)
}
