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

func list[T any](x ...T) []T { return x }

func TestEngine_Class(t *testing.T) {
	domain := mock.Domain("mock")
	foo, bar := domain.Class("foo"), domain.Class("bar")
	rule := mock.NewRule("x", list(foo), list(bar), mock.NewQuery(bar, ""))
	for _, x := range []struct {
		name string
		want korrel8r.Class
		err  string
	}{
		{"mock:foo", foo, ""},
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
			g := e.Graph().ShortestPaths(a, goals...)
			ctx := korrel8r.WithConstraint(context.Background(), x.constraint)
			g, err := traverse.NewSync(e, g).Goals(ctx, traverse.Start{Class: a, Objects: []korrel8r.Object{obj{"a", ontime}}}, goals)
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
