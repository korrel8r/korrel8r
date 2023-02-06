package engine

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestEngine_Follow(t *testing.T) {
	// FIXME
	// s := mock.Store{}
	// path := graph.MultiPath{
	// 	graph.Links{
	// 		// Return 2 results, must follow both
	// 		mock.NewRule("ab", "a", "b", func(korrel8r.Object, *korrel8r.Constraint) (korrel8r.Query, error) {
	// 			return s.NewQuery("b:1", "b:2"), nil
	// 		}),
	// 	},
	// 	graph.Links{
	// 		// 2 rules, must follow both. Incorporate data from stat object.
	// 		mock.NewRule("bc", "b", "c", func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
	// 			return s.NewQuery("c:" + start.(mock.Object).Data()), nil
	// 		}),
	// 		mock.NewRule("bc2", "b", "c", func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
	// 			return s.NewQuery("c:x" + start.(mock.Object).Data()), nil
	// 		}),
	// 	},
	// 	graph.Links{
	// 		mock.NewRule("cz", "c", "z", func(start korrel8r.Object, _ *korrel8r.Constraint) (korrel8r.Query, error) {
	// 			return s.NewQuery("z:" + start.(mock.Object).Data()), nil
	// 		}),
	// 	},
	// }
	// want := NewResult(mock.Class("z"))
	// want.Queries.Append(s.NewQuery("z:1"), s.NewQuery("z:2"), s.NewQuery("z:x1"), s.NewQuery("z:x2"))
	// e := New()
	// e.AddDomain(mock.Domain(""), s)
	// var results Results
	// e.Follow(context.Background(), mock.Objects("foo:a"), nil, path, &results)
	// if assert.NotEmpty(t, results) {
	// 	assert.ElementsMatch(t, want.Queries.List, results[len(results)-1].Queries.List)
	// }
}
