// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"slices"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataIdentityAndGraphOwnership(t *testing.T) {
	b := mock.NewBuilder("d")
	rule := b.Rule("ab", "d:a", "d:b", nil)
	input := []korrel8r.Rule{rule}
	d := NewData(input...)
	input[0] = nil // Data owns its rule table, not the caller's slice.
	require.Same(t, rule, d.RuleForLine(0))

	a := b.Class("d:a")
	aID, ok := d.NodeID(a)
	require.True(t, ok)
	assert.Equal(t, a, d.Class(aID))
	assert.Equal(t, 2, d.NodeCount())
	_, ok = d.NodeID(b.Class("d:absent"))
	assert.False(t, ok)
	classes := d.Classes()
	classes[aID] = nil
	assert.Equal(t, a, d.Class(aID), "class table must not alias returned slices")
	empty := New(d)
	assert.Nil(t, empty.NodeFor(a), "Data membership is not Graph membership")
	assert.Empty(t, empty.LineStrings())

	first, second := d.FullGraph(), d.FullGraph()
	n := first.NodeFor(a)
	assert.Equal(t, aID, n.ID())
	assert.NotSame(t, n, second.NodeFor(a))
	n.Result.Append(1)
	assert.True(t, second.NodeFor(a).Empty())
	query := b.Query("d:b", "q")
	l := first.FindLine(a, b.Class("d:b"), rule)
	l.Queries.Set(query, 7)
	l.Attrs["label"] = "search result"
	assert.Same(t, n, l.Start())
	assert.Empty(t, second.FindLine(a, b.Class("d:b"), rule).Queries)

	selected := first.Select(func(*Line) bool { return true })
	copy := selected.FindLine(a, b.Class("d:b"), rule)
	assert.Equal(t, l.ID(), copy.ID())
	assert.NotSame(t, l, copy)
	assert.Empty(t, copy.Queries, "Select copies topology, not query accounting")
	assert.Empty(t, copy.Attrs)
	assert.Same(t, selected.NodeFor(a), copy.Start())

	view := d.Graph()
	assert.Equal(t, aID, view.Node(aID).ID())
	_, mutable := view.Node(aID).(*Node)
	assert.False(t, mutable, "view nodes are IDs, not mutable result nodes")
	from, to := d.Endpoints(0)
	freshFrom, freshTo := d.NewNode(from), d.NewNode(to)
	fresh := d.NewLine(0, freshFrom, freshTo)
	fresh.Queries.Set(query, 3)
	assert.Empty(t, d.NewLine(0, freshFrom, freshTo).Queries)
	assert.Same(t, freshFrom, fresh.Start())
	assert.Same(t, freshTo, fresh.Goal())
	assert.NotSame(t, freshFrom, d.NewNode(from))
	assert.Panics(t, func() { d.NewLine(0, freshTo, freshFrom) })
	assert.Panics(t, func() { d.NewLine(0, nil, freshTo) })
}

func TestDataRules(t *testing.T) {
	b := mock.NewBuilder("d")
	wide := b.Rule("wide", "d:a", []string{"d:b", "d:c"}, nil)
	empty := b.Rule("empty", "d:a", []string{}, nil)
	d := NewData(wide, empty, wide)
	assert.Equal(t, 4, d.LineCount())
	rules := d.Rules()
	assert.Equal(t, []korrel8r.Rule{wide, empty, wide}, rules)
	rules[0] = nil
	assert.Equal(t, []korrel8r.Rule{wide, empty, wide}, d.Rules(), "returned rules must not alias Data")

	selected := slices.DeleteFunc(d.Rules(), func(r korrel8r.Rule) bool { return r != wide })
	g := NewData(selected...).FullGraph()
	want := d.FullGraph().Select(func(l *Line) bool { return l.Rule == wide })
	assert.Equal(t, want.LineStrings(), g.LineStrings())
	assert.Equal(t, want.NodeStrings(true), g.NodeStrings(true))
}

func TestDataExpansionOrder(t *testing.T) {
	b := mock.NewBuilder("d")
	rule := b.Rule("a", "d:a", []string{"d:a", "d:b"}, nil)
	d := NewData(rule, rule)
	assert.Equal(t, 4, d.LineCount())
	assert.Equal(t, []korrel8r.Rule{rule, rule}, d.Rules())
	assert.Equal(t, []korrel8r.Class{b.Class("d:a"), b.Class("d:b")}, d.Classes())
	for id := 0; id < d.LineCount(); id++ {
		from, to := d.Endpoints(id)
		assert.Equal(t, int64(0), from)
		assert.Equal(t, int64(id%2), to)
		assert.Equal(t, int64(id), d.NewLine(id, d.NewNode(from), d.NewNode(to)).ID())
	}
}
