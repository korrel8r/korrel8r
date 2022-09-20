package loki

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Query_String(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)
	lines := []string{"hello", "there", "mr. frog"}
	err := l.Push(map[string]string{"test": "loki"}, lines...)
	require.NoError(t, err)

	s, err := NewStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	var want []korrel8.Object
	for _, l := range lines {
		want = append(want, Object(l))
	}
	got, err := s.Query(context.Background(), `{test="loki"}`)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestStore_Query_QueryObject(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)

	err := l.Push(map[string]string{"test": "loki"}, "much", "too", "early")
	require.NoError(t, err)

	t1 := time.Now()
	err = l.Push(map[string]string{"test": "loki"}, "right", "on", "time")
	require.NoError(t, err)
	t2 := time.Now()

	err = l.Push(map[string]string{"test": "loki"}, "much", "too", "late")
	require.NoError(t, err)

	s, err := NewStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	for _, x := range []struct {
		query string
		want  []korrel8.Object
	}{
		{
			query: QueryObject{End: &t1, Query: `{test="loki"}`}.String(),
			want:  []korrel8.Object{Object("much"), Object("too"), Object("early")},
		},
		{
			query: QueryObject{Start: &t1, End: &t2, Query: `{test="loki"}`}.String(),
			want:  []korrel8.Object{Object("right"), Object("on"), Object("time")},
		},
		{
			query: QueryObject{Start: &t2, Query: `{test="loki"}`}.String(),
			want:  []korrel8.Object{Object("much"), Object("too"), Object("late")},
		},
		{
			query: QueryObject{Start: &t1, End: &t2, Direction: "backward", Query: `{test="loki"}`}.String(),
			want:  []korrel8.Object{Object("time"), Object("on"), Object("right")},
		},
		{
			query: QueryObject{Limit: 5, Query: `{test="loki"}`}.String(),
			want:  []korrel8.Object{Object("much"), Object("too"), Object("early"), Object("right"), Object("on")},
		},
		// FIXME more cases for all fields.
	} {
		t.Run(x.query, func(t *testing.T) {
			got, err := s.Query(context.Background(), x.query)
			assert.NoError(t, err)
			assert.Equal(t, x.want, got)
		})
	}
}

// FIXME more tests: multi-stream results sorted correctly, QueryObject fields respected.
