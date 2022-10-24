package loki

import (
	"context"
	"net/http"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLokiStore_Get(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)
	lines := []string{"hello", "there", "mr. frog"}
	err := l.Push(map[string]string{"test": "loki"}, lines...)
	require.NoError(t, err)
	s, err := NewLokiStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	var want []korrel8.Object
	for _, l := range lines {
		want = append(want, Object(l))
	}
	query := Query(`{test="loki"}`)
	result := korrel8.NewListResult()
	require.NoError(t, s.Get(ctx, &query, result))
	assert.Equal(t, want, result.List())
}

// FIXME constraints
// func TestStore_Query_QueryConstraint(t *testing.T) {
// 	t.Parallel()
// 	l := test.RequireLokiServer(t)

// 	err := l.Push(map[string]string{"test": "loki"}, "much", "too", "early")
// 	require.NoError(t, err)

// 	t1 := time.Now()
// 	err = l.Push(map[string]string{"test": "loki"}, "right", "on", "time")
// 	require.NoError(t, err)
// 	t2 := time.Now()

// 	err = l.Push(map[string]string{"test": "loki"}, "much", "too", "late")
// 	require.NoError(t, err)
// 	s, err := NewStore(l.URL(), http.DefaultClient)
// 	require.NoError(t, err)

// 	for n, x := range []struct {
// 		query korrel8.Query
// 		want  []korrel8.Object
// 	}{
// 		{
// 			query: &Query{Query: `{test="loki"}`, Constraint: &korrel8.Constraint{End: &t1}},
// 			want:  []korrel8.Object{Object("much"), Object("too"), Object("early")},
// 		},
// 		{
// 			query: &Query{Query: `{test="loki"}`, Constraint: &korrel8.Constraint{Start: &t1, End: &t2}},
// 			want:  []korrel8.Object{Object("right"), Object("on"), Object("time")},
// 		},
// 	} {
// 		t.Run(strconv.Itoa(n), func(t *testing.T) {
// 			var result korrel8.ListResult
// 			assert.NoError(t, s.Get(ctx, x.query, &result))
// 			assert.Equal(t, x.want, result.List())
// 		})
// 	}
// }

var ctx = context.Background()
