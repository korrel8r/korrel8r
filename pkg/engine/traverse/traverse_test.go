// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse_test

import (
	"context"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Clean up and make test more concise and readable.

func TestFollower_Traverse(t *testing.T) {
	d := mock.Domain("mock")
	s := mock.NewStore(d)
	c := d.Class
	ca, cb, cc, cz := c("a"), c("b"), c("c"), c("z")
	r := func(name string, start, goal korrel8r.Class, result ...any) korrel8r.Rule {
		return mock.NewRule(name, list(start), list(goal), result...)
	}

	e, err := engine.Build().Rules(
		// Return 2 results, must follow both
		r("ab", ca, cb, s, cb, 1, 2),
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

	for _, x := range []struct {
		name string
		t    traverse.Traverser
	}{
		{name: "sync", t: traverse.NewSync(e, e.Graph())},
		{name: "async", t: traverse.NewAsync(e, e.Graph())},
	} {
		// FIXME neighbour tests also
		t.Run(x.name, func(t *testing.T) {
			start := traverse.Start{Class: ca, Objects: []korrel8r.Object{0}}
			g, err := x.t.Goals(context.Background(), start, list(cz))
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
		})
	}
}

// FIXME more tests

func list[T any](x ...T) []T { return x }