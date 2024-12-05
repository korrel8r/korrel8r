// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Class(t *testing.T) {
	domain := mock.Domain("mock")
	rule := mock.NewRuleQuery(domain.Name(), domain.Class("foo"), domain.Class("bar"), nil)
	for _, x := range []struct {
		name string
		want korrel8r.Class
		err  string
	}{
		{"mock:foo", domain.Class("foo"), ""},
		{"x.", nil, "invalid class name: x."},
		{".x", nil, "invalid class name: .x"},
		{"x", nil, "invalid class name: x"},
		{"", nil, "invalid class name: "},
		{"bad:foo", nil, `domain not found: "bad"`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e, err := engine.Build().Rules(rule).Engine()
			require.NoError(t, err)
			assert.Equal(t, []korrel8r.Domain{domain}, e.Domains())
			c, err := e.Class(x.name)
			if x.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, x.err)
			}
			assert.Equal(t, x.want, c)
		})
	}
}

func TestFollower_Traverse(t *testing.T) {
	d := mock.Domain("mock")
	s := mock.NewStore(d)
	a, b, c, z := d.Class("a"), d.Class("b"), d.Class("c"), d.Class("z")
	e, err := engine.Build().Rules(
		// Return 2 results, must follow both
		mock.NewRuleQuery("ab", a, b, s.NewQuery(b, 1, 2)),
		// 2 rules, must follow both. Incorporate data from start object.
		mock.NewRule("bc1", []korrel8r.Class{b}, []korrel8r.Class{c}, func(start korrel8r.Object) (korrel8r.Query, error) {
			return s.NewQuery(c, start), nil
		}),
		mock.NewRule("bc2", []korrel8r.Class{b}, []korrel8r.Class{c}, func(start korrel8r.Object) (korrel8r.Query, error) {
			return s.NewQuery(c, start.(int)+10), nil
		}),
		mock.NewRule("cz", []korrel8r.Class{c}, []korrel8r.Class{z}, func(start korrel8r.Object) (korrel8r.Query, error) {
			return s.NewQuery(z, start), nil
		}),
	).Stores(s).Engine()
	require.NoError(t, err)
	g, err := traverse.NewSync(e, e.Graph(), a, []korrel8r.Object{0}, nil).Goals(context.Background(), []korrel8r.Class{z})
	assert.NoError(t, err)
	// Check node results
	assert.ElementsMatch(t, []korrel8r.Object{0}, g.NodeFor(a).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2}, g.NodeFor(b).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(c).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(z).Result.List())
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

func (o obj) String() string { return o.Name }

func TestEngine_PropagateConstraints(t *testing.T) {
	d := mock.Domain("mock")
	a, b, c := d.Class("a"), d.Class("b"), d.Class("c")
	// Time range [start,end] and some time points.
	start := time.Now()
	end := start.Add(time.Minute)
	afterEnd := end.Add(time.Minute)
	early, ontime, late := start.Add(-1), start.Add(1), end.Add(1)
	s := mock.NewStore(d)

	s.ConstraintFunc = func(c *korrel8r.Constraint, o korrel8r.Object) bool {
		return c.CompareTime(o.(obj).Time) == 0
	}

	// Rules to test constraints.
	e, err := engine.Build().Rules(
		// Generate objects with timepoints.
		mock.NewRule("ab", []korrel8r.Class{a}, []korrel8r.Class{b}, func(korrel8r.Object) (korrel8r.Query, error) {
			return s.NewQuery(b, obj{"x", early}, obj{"y", ontime}, obj{"z", late}), nil
		}),
		// Generate objects with timeponts and record name of previous object.
		mock.NewRule("bc", []korrel8r.Class{b}, []korrel8r.Class{c}, func(o korrel8r.Object) (korrel8r.Query, error) {
			name := o.(obj).Name
			return s.NewQuery(c, obj{"u" + name, early}, obj{"v" + name, ontime}, obj{"w" + name, late}), nil
		})).Stores(s).Engine()
	require.NoError(t, err)

	// Test traversals, verify constraints are applied.
	for i, x := range []struct {
		constraint *korrel8r.Constraint
		want       []obj
	}{
		{&korrel8r.Constraint{Start: &start, End: &end}, []obj{
			{"vy", ontime},
		}},
		{&korrel8r.Constraint{End: &afterEnd}, []obj{
			{"ux", early}, {"vx", ontime}, {"wx", late},
			{"uy", early}, {"vy", ontime}, {"wy", late},
			{"uz", early}, {"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{Start: &start, End: &afterEnd}, []obj{
			{"vy", ontime}, {"wy", late},
			{"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{End: &end}, []obj{
			{"ux", early}, {"vx", ontime},
			{"uy", early}, {"vy", ontime},
		}},
	} {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			goals := []korrel8r.Class{c}
			g := e.Graph().AllPaths(a, goals...)
			ctx := korrel8r.WithConstraint(context.Background(), x.constraint)
			g, err := traverse.NewSync(e, g, a, []korrel8r.Object{obj{"a", ontime}}, nil).Goals(ctx, goals)
			assert.NoError(t, err)
			got := g.NodeFor(c).Result.List()
			assert.Equal(t, asStrings(x.want), asStrings(got), "want %v got %v", x.want, got)
		})
	}
}

func TestEngine_ConfigMockStore(t *testing.T) {
	d := mock.Domain("mock")
	e, err := engine.Build().Domains(d).ConfigFile("testdata/korrel8r.yaml").Engine()
	require.NoError(t, err)
	q, err := e.Query("mock:foo:hello")
	require.NoError(t, err)
	r := graph.NewResult(q.Class())
	require.NoError(t, e.Get(context.Background(), q, nil, r))
	assert.Equal(t, []korrel8r.Object{"hello", "there"}, r.List())
}

func asStrings[T any](v []T) []string {
	s := make([]string, len(v))
	for i := range v {
		s[i] = fmt.Sprint(v[i])
	}
	slices.Sort(s)
	return s
}

func TestEngineStoreFor(t *testing.T) {
	d := mock.Domain("mock")
	// add a second mock store, verify both stores are checked by Get.
	s := mock.NewStore(d)
	q := mock.NewQuery(d.Class("foo"), "hello")
	s.Add(mock.QueryMap{q.String(): []korrel8r.Object{"dolly"}})
	e, err := engine.Build().Domains(d).ConfigFile("testdata/korrel8r.yaml").Stores(s).Engine()
	require.NoError(t, err)

	r := graph.NewListResult()
	require.NoError(t, e.StoreFor(d).Get(context.Background(), q, nil, r))
	assert.ElementsMatch(t, []korrel8r.Object{"hello", "there", "dolly"}, r.List())
}

// Mock object has a name and a timestamp.
type obj struct {
	Name string
	Time time.Time
}
