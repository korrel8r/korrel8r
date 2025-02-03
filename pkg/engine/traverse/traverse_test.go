// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"
	"errors"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Clean up and make test more concise and readable.

func r(name string, start, goal korrel8r.Class, apply any) korrel8r.Rule {
	return mock.NewRule(name, list(start), list(goal), apply)
}

func TestTraverserGoals(t *testing.T) {
	d := mock.Domain("mock")
	s := mock.NewStore(d)
	c := d.Class
	ca, cb, cc, cz := c("a"), c("b"), c("c"), c("z")

	e, err := engine.Build().Rules(
		// Return 2 results, must follow both
		r("ab", ca, cb, mock.NewQuery(cb, "1", 1, 2)),
		// 2 rules, must follow both. Incorporate data from start object.
		r("bc1", cb, cc, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(cc, test.JSONString(start), start), nil
		}),
		r("bc2", cb, cc, func(start korrel8r.Object) (korrel8r.Query, error) {
			result := start.(int) + 10
			return mock.NewQuery(cc, test.JSONString(result), result), nil
		}),
		r("cz", cc, cz, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(cz, test.JSONString(start), start), nil
		}),
	).Stores(s).Engine()
	require.NoError(t, err)

	a := NewAsync(e, e.Graph())
	start := Start{Class: ca, Objects: []korrel8r.Object{0}}
	g, err := a.Goals(context.Background(), start, list(cz))
	assert.NoError(t, err)
	// Check node results
	assert.ElementsMatch(t, []korrel8r.Object{0}, g.NodeFor(ca).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2}, g.NodeFor(cb).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(cc).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(cz).Result.List())
	// Check line results
	g.EachLine(func(l *graph.Line) {
		switch l.Rule.Name() {
		case "ab":
			q, err := l.Rule.Apply(0)
			require.NoError(t, err)
			assert.Equal(t, 2, l.Queries.Get(q), q.String())
			assert.Len(t, l.Queries, 1)
		case "bc1", "bc2":
			q1, err := l.Rule.Apply(1)
			require.NoError(t, err)
			q2, err := l.Rule.Apply(2)
			require.NoError(t, err)
			assert.Len(t, l.Queries, 2)
			assert.Equal(t, 1, l.Queries.Get(q1))
			assert.Equal(t, 1, l.Queries.Get(q2))
		case "cz":
			q1, err := l.Rule.Apply(1)
			require.NoError(t, err)
			q2, err := l.Rule.Apply(2)
			require.NoError(t, err)
			q3, err := l.Rule.Apply(11)
			require.NoError(t, err)
			q4, err := l.Rule.Apply(12)
			require.NoError(t, err)
			assert.Len(t, l.Queries, 4)
			assert.Equal(t, 1, l.Queries.Get(q1), q1.String())
			assert.Equal(t, 1, l.Queries.Get(q2), q2.String())
			assert.Equal(t, 1, l.Queries.Get(q3), q3.String())
			assert.Equal(t, 1, l.Queries.Get(q4), q4.String())
		default:
			t.Fatalf("unexpected rule: %v", l.Rule)
		}
	})
}

func TestTraverserNeighbours(t *testing.T) {
	d := mock.Domain("mock")
	s := mock.NewStore(d)
	c := d.Class
	ca, cb, cc, cz := c("a"), c("b"), c("c"), c("z")

	e, err := engine.Build().Rules(
		// Return 2 results, must follow both
		r("ab", ca, cb, mock.NewQuery(cb, "1", 1, 2)),
		// 2 rules, must follow both. Incorporate data from start object.
		r("bc1", cb, cc, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(cc, test.JSONString(start), start), nil
		}),
		r("bc2", cb, cc, func(start korrel8r.Object) (korrel8r.Query, error) {
			result := start.(int) + 10
			return mock.NewQuery(cc, test.JSONString(result), result), nil
		}),
		r("cz", cc, cz, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(cz, test.JSONString(start), start), nil
		}),
	).Stores(s).Engine()
	require.NoError(t, err)

	a := NewAsync(e, e.Graph())
	start := Start{Class: ca, Objects: []korrel8r.Object{0}}
	g, err := a.Neighbours(context.Background(), start, 2)
	assert.NoError(t, err)
	// Check node results
	assert.ElementsMatch(t, []korrel8r.Object{0}, g.NodeFor(ca).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2}, g.NodeFor(cb).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(cc).Result.List())
	// Check line results
	g.EachLine(func(l *graph.Line) {
		switch l.Rule.Name() {
		case "ab":
			q, err := l.Rule.Apply(0)
			require.NoError(t, err)
			assert.Equal(t, 2, l.Queries.Get(q), q.String())
			assert.Len(t, l.Queries, 1)
		case "bc1", "bc2":
			q1, err := l.Rule.Apply(1)
			require.NoError(t, err)
			q2, err := l.Rule.Apply(2)
			require.NoError(t, err)
			assert.Len(t, l.Queries, 2)
			assert.Equal(t, 1, l.Queries.Get(q1))
			assert.Equal(t, 1, l.Queries.Get(q2))
		case "cz":
			q1, err := l.Rule.Apply(1)
			require.NoError(t, err)
			q2, err := l.Rule.Apply(2)
			require.NoError(t, err)
			q3, err := l.Rule.Apply(11)
			require.NoError(t, err)
			q4, err := l.Rule.Apply(12)
			require.NoError(t, err)
			assert.Len(t, l.Queries, 4)
			assert.Equal(t, 1, l.Queries.Get(q1), q1.String())
			assert.Equal(t, 1, l.Queries.Get(q2), q2.String())
			assert.Equal(t, 1, l.Queries.Get(q3), q3.String())
			assert.Equal(t, 1, l.Queries.Get(q4), q4.String())
		default:
			t.Fatalf("unexpected rule: %v", l.Rule)
		}
	})
}

// FIXME TestTraverserNeighbours

// Make sure we skip rules with unknown classes and get results.
func TestTraverser_skip_unknown_classes(t *testing.T) {
	d := mock.Domain("mock")
	c := d.Class
	ca, cb, cc := c("a"), c("b"), c("c")
	s := mock.NewStore(d, ca, cb)

	e, err := engine.Build().Rules(
		r("ab", ca, cb, mock.NewQuery(cb, "1", 1)),
		r("bc", cb, cc, mock.NewQuery(cb, "1", 1)),
	).Stores(s).Engine()
	require.NoError(t, err)

	a := NewAsync(e, e.Graph())
	start := Start{Class: ca, Objects: []korrel8r.Object{0}}
	g, err := a.Neighbours(context.Background(), start, 2)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []korrel8r.Object{0}, g.NodeFor(ca).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1}, g.NodeFor(cb).Result.List())
	assert.Equal(t, 2, g.Nodes().Len())
}

func TestPartialError(t *testing.T) {
	d := mock.Domain("mock")
	s := mock.NewStore(d)
	c := d.Class
	ca, cb, cc := c("a"), c("b"), c("c")
	e, err := engine.Build().Rules(
		r("ab", ca, cb, mock.NewQuery(cb, "1,2", 1, 2)),
		r("bc", cb, cc, mock.NewQueryError(cc, "err", errors.New("no good"))),
	).Stores(s).Engine()
	require.NoError(t, err)
	start := Start{Class: ca, Objects: []korrel8r.Object{0}}
	g, err := New(e, e.Graph()).Neighbours(context.Background(), start, 3)
	var pe *PartialError
	assert.ErrorContains(t, err, "no good")
	assert.ErrorAs(t, err, &pe)
	assert.ElementsMatch(t, g.NodesFor(ca, cb, cc), graph.NodesOf(g.Nodes()))
	assert.Equal(t, []any{0}, g.NodeFor(ca).Result.List())
	assert.Equal(t, []any{1, 2}, g.NodeFor(cb).Result.List())
	assert.Empty(t, g.NodeFor(cc).Result.List())
}

func TestErrors(t *testing.T) {
	assert.NoError(t, NewErrors().Err())

	errs := NewErrors()
	errs.Add(errors.New("bad"))
	assert.EqualError(t, errs.Err(), "bad")
	errs.Add(errors.New("worse"))
	assert.EqualError(t, errs.Err(), "bad\nworse")
	errs.Add(errors.New("bad")) // Don't repeat
	assert.EqualError(t, errs.Err(), "bad\nworse")
	assert.False(t, IsPartial(errs.Err()))
	errs.Add(nil) // Partial success
	assert.True(t, IsPartial(errs.Err()))
	assert.ErrorContains(t, errs.Err(), "bad\nworse")
}

func list[T any](x ...T) []T { return x }
