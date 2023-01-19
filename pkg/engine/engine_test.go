package engine

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
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
		{"bad/foo", nil, `domain not found: bad`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := New()
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
	s := mock.Store{}
	path := graph.MultiPath{
		graph.Links{
			// Return 2 results, must follow both
			mock.NewRule("ab", "a", "b", func(korrel8.Object, *korrel8.Constraint) (uri.Reference, error) {
				return s.NewReference("b:1", "b:2"), nil
			}),
		},
		graph.Links{
			// 2 rules, must follow both. Incorporate data from stat object.
			mock.NewRule("bc", "b", "c", func(start korrel8.Object, _ *korrel8.Constraint) (uri.Reference, error) {
				return s.NewReference("c:" + start.(mock.Object).Data()), nil
			}),
			mock.NewRule("bc2", "b", "c", func(start korrel8.Object, _ *korrel8.Constraint) (uri.Reference, error) {
				return s.NewReference("c:x" + start.(mock.Object).Data()), nil
			}),
		},
		graph.Links{
			mock.NewRule("cz", "c", "z", func(start korrel8.Object, _ *korrel8.Constraint) (uri.Reference, error) {
				return s.NewReference("z:" + start.(mock.Object).Data()), nil
			}),
		},
	}
	want := NewResult(mock.Class("z"))
	want.References.Append(s.NewReference("z:1"), s.NewReference("z:2"), s.NewReference("z:x1"), s.NewReference("z:x2"))
	e := New()
	e.AddDomain(mock.Domain(""), s)
	results := NewResults()
	err := e.Follow(context.Background(), mock.Objects("foo:a"), nil, path, results)
	assert.NoError(t, err)
	last := results.List[len(results.List)-1]
	assert.ElementsMatch(t, want.References.List, last.References.List)
}
