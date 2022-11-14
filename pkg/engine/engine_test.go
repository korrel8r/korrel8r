package engine

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_ParseClass(t *testing.T) {
	for _, x := range []struct {
		name string
		want korrel8.Class
		err  string
	}{
		{"mock/foo", mock.Domain("mock").Class("foo"), ""},
		{"x/", nil, "invalid class name: x/"},
		{"/x", nil, "invalid class name: /x"},
		{"x", nil, "invalid class name: x"},
		{"", nil, "invalid class name: "},
		{"bad/foo", nil, `unknown domain in class name: bad/foo`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := New("")
			e.AddDomain(mock.Domain("mock"), nil)
			c, err := e.ParseClass(x.name)
			if x.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, x.err)
			}
			assert.Equal(t, x.want, c)
		})
	}
}

func TestEngine_Follow(t *testing.T) {
	path := graph.MultiPath{
		graph.Links{
			// Return 2 results, must follow both
			mock.NewRule("ab", "a", "b", func(korrel8.Object, *korrel8.Constraint) (*korrel8.Query, error) {
				return mock.NewQuery("b:1", "b:2"), nil
			}),
		},
		graph.Links{
			mock.NewRule("bc", "b", "c", func(start korrel8.Object, _ *korrel8.Constraint) (*korrel8.Query, error) {
				return mock.NewQuery(mock.Object("c:" + start.(mock.Object).Data())), nil
			}),
			mock.NewRule("bc2", "b", "c", func(start korrel8.Object, _ *korrel8.Constraint) (*korrel8.Query, error) {
				return mock.NewQuery(mock.Object("c:x" + start.(mock.Object).Data())), nil
			}),
		},
		graph.Links{
			mock.NewRule("cz", "c", "z", func(start korrel8.Object, _ *korrel8.Constraint) (*korrel8.Query, error) {
				return mock.NewQuery(mock.Object("z:" + start.(mock.Object).Data())), nil
			}),
		},
	}
	want := []korrel8.Query{*mock.NewQuery("z:1"), *mock.NewQuery("z:2"), *mock.NewQuery("z:x1"), *mock.NewQuery("z:x2")}
	e := New("")
	e.AddDomain(mock.Domain(""), mock.Store{})
	queries, err := e.Follow(context.Background(), []korrel8.Object{mock.Object("foo:a")}, nil, path)
	assert.NoError(t, err)
	assert.ElementsMatch(t, want, queries)
}
