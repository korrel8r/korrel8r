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
	for _, x := range []struct {
		name string
		want korrel8r.Class
		err  string
	}{
		{"mock/foo", mock.Domain("mock").Class("foo"), ""},
		{"x/", nil, "invalid class name: x/"},
		{"/x", nil, "invalid class name: /x"},
		{"x", nil, "invalid class name: x"},
		{"", nil, "invalid class name: "},
		{"bad/foo", nil, `domain not found: bad`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := New()
			e.AddDomain(mock.Domain("mock"), nil)
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
	domains := []mock.Domain{"a", "b", "c"}
	var want []korrel8r.Domain
	e := New()
	for _, d := range domains {
		e.AddDomain(d, nil)
		want = append(want, d)
	}
	assert.ElementsMatch(t, want, maps.Values(e.Domains()))

}

// FIXME improved mock for all.
func TestFollower_Traverse(t *testing.T) {
	s := mock.Store{}
	e := New()
	e.AddDomain(mock.Domain(""), s)
	e.AddRules(
		// Return 2 results, must follow both
		mock.NewRule("ab", "a", "b", func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error) {
			return s.NewQuery("b:1", "b:2"), nil
		}),
		// 2 rules, must follow both. Incorporate data from stat object.
		mock.NewRule("bc", "b", "c", func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
			return s.NewQuery("c:" + start.(mock.Object).Data()), nil
		}),
		mock.NewRule("bc2", "b", "c", func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
			return s.NewQuery("c:x" + start.(mock.Object).Data()), nil
		}),
		mock.NewRule("cz", "c", "z", func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
			return s.NewQuery("z:" + start.(mock.Object).Data()), nil
		}))
	g := e.Graph()
	g.NodeFor(mock.Class("a")).Result.Append(mock.Objects("a:0")...)
	f := e.Follower(context.Background())
	assert.NoError(t, g.Traverse(f.Traverse))
	assert.NoError(t, f.Err)
	for _, x := range []struct {
		class string
		data  []korrel8r.Object
	}{
		{"a", mock.Objects("a:0")},
		{"b", mock.Objects("b:1", "b:2")},
		{"c", mock.Objects("c:1", "c:2", "c:x1", "c:x2")},
		{"z", mock.Objects("z:1", "z:2", "z:x1", "z:x2")},
	} {
		assert.ElementsMatch(t, x.data, g.NodeFor(mock.Class(x.class)).Result.List())
	}
}
