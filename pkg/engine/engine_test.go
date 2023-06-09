// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestEngine_Class(t *testing.T) {
	domain := mock.Domain("mock")
	for _, x := range []struct {
		name string
		want korrel8r.Class
		err  string
	}{
		{"mock/foo", domain.Class("foo"), ""},
		{"x/", nil, "invalid class name: x/"},
		{"/x", nil, "invalid class name: /x"},
		{"x", nil, "invalid class name: x"},
		{"", nil, "invalid class name: "},
		{"bad/foo", nil, `domain not found: bad`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := New(domain)
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

func TestEngine_Domains(t *testing.T) {
	domains := []korrel8r.Domain{mock.Domain("a"), mock.Domain("b"), mock.Domain("c")}
	e := New(domains...)
	assert.ElementsMatch(t, domains, maps.Values(e.Domains()))
}

func TestFollower_Traverse(t *testing.T) {
	d := mock.Domain("mock")
	s := mock.NewStore(d)
	e := New(d)
	a, b, c, z := d.Class("a"), d.Class("b"), d.Class("c"), d.Class("z")
	require.NoError(t, e.AddStore(s))
	e.AddRules(
		// Return 2 results, must follow both
		mock.NewApplyRule(a, b, func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error) {
			return mock.NewQuery(b, 1, 2), nil
		}),
		// 2 rules, must follow both. Incorporate data from start object.
		mock.NewApplyRule(b, c, func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
			return mock.NewQuery(c, start), nil
		}),
		mock.NewApplyRule(b, c, func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
			return mock.NewQuery(c, start.(int)+10), nil
		}),
		mock.NewApplyRule(c, z, func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
			return mock.NewQuery(z, start), nil
		}))
	g := e.Graph()
	g.NodeFor(a).Result.Append(0)
	f := e.Follower(context.Background())
	assert.NoError(t, g.Traverse(f.Traverse))
	assert.NoError(t, f.Err)
	assert.ElementsMatch(t, []korrel8r.Object{0}, g.NodeFor(a).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2}, g.NodeFor(b).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(c).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(z).Result.List())
}
