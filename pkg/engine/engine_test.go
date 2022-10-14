package engine

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Parse(t *testing.T) {
	for _, x := range []struct {
		name string
		want korrel8.Class
		err  string
	}{
		{"mock/foo", mockDomain{}.Class("foo"), ""},
		{"mock", mockDomain{}.Class(""), ""}, // Allow "default" class, empty string
		{"nosuch", nil, `unknown domain: "nosuch"`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := New()
			e.AddDomain(mockDomain{}, nil)
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
	path := korrel8.Path{
		// Return 2 results, must follow both
		rr("a", "b", func(korrel8.Object, *korrel8.Constraint) korrel8.Query { return korrel8.Query("1.b,2.b") }),
		// Replace start object's class with goal class
		rr("b", "c", func(start korrel8.Object, _ *korrel8.Constraint) korrel8.Query {
			return korrel8.Query(start.(mockObject).name + ".c")
		}),
		rr("c", "z", func(start korrel8.Object, _ *korrel8.Constraint) korrel8.Query {
			return korrel8.Query(start.(mockObject).name + ".z")
		}),
	}
	want := []korrel8.Query{"1.z", "2.z"}

	e := New()
	e.AddDomain(mockDomain{}, mockStore{})
	queries, err := e.Follow(context.Background(), o("foo", "a"), nil, path)
	assert.NoError(t, err)
	assert.ElementsMatch(t, want, queries)
}
