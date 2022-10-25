package engine

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
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
		{"mock/foo", mock.Domain{}.Class("foo"), ""},
		{"mock", mock.Domain{}.Class(""), ""}, // Allow "default" class, empty string
		{"nosuch", nil, `unknown domain: "nosuch"`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e := New()
			e.AddDomain(mock.Domain{}, nil)
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
	path := []korrel8.Rule{
		// Return 2 results, must follow both
		mock.NewRule("a", "b", func(korrel8.Object, *korrel8.Constraint) *korrel8.Query {
			return &korrel8.Query{Path: "1.b/2.b"}
		}),
		// Replace start object's class with goal class
		mock.NewRule("b", "c", func(start korrel8.Object, _ *korrel8.Constraint) *korrel8.Query {
			return &korrel8.Query{Path: start.(mock.Object).Name + ".c"}
		}),
		mock.NewRule("c", "z", func(start korrel8.Object, _ *korrel8.Constraint) *korrel8.Query {
			return &korrel8.Query{Path: start.(mock.Object).Name + ".z"}
		}),
	}
	want := []korrel8.Query{{Path: "1.z"}, {Path: "2.z"}}

	e := New()
	e.AddDomain(mock.Domain{}, mock.Store{})
	queries, err := e.Follow(context.Background(), []korrel8.Object{mock.NewObject("foo", "a")}, nil, path)
	assert.NoError(t, err)
	assert.ElementsMatch(t, want, queries)
}
