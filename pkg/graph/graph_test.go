// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTwoPaths(t *testing.T) {
	b := mock.NewBuilder("d")
	r := b.Rule
	// Two independent paths to c:
	// a->b->c
	// a->x->c
	g := NewData(
		r("ab", "d:a", "d:b", b.Query("d:b", "ab", 1)),
		r("bc", "d:b", "d:c", b.Query("d:c", "bc", 2)),
		r("ac", "d:a", "d:c", b.Query("d:x", "ac", 3)),
		r("ax", "d:a", "d:x", b.Query("d:x", "ax", 4)),
	).FullGraph()

	sub := g.Select(func(l *Line) bool { return true })
	assert.ElementsMatch(t, []string{
		"ab(d:a->d:b)",
		"bc(d:b->d:c)",
		"ac(d:a->d:c)",
		"ax(d:a->d:x)",
	}, sub.LineStrings())
	assert.ElementsMatch(t, []string{
		"d:a",
		"d:b",
		"d:c",
		"d:x",
	}, sub.NodeStrings(true))
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
