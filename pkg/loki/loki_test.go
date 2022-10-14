package loki

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Query_String(t *testing.T) {
	t.Parallel()
	l := test.RequireLokiServer(t)
	lines := []string{"hello", "there", "mr. frog"}
	err := l.Push(map[string]string{"test": "loki"}, lines...)
	require.NoError(t, err)
	s := NewStore(l.URL(), http.DefaultClient)

	var want []korrel8.Object
	for _, l := range lines {
		want = append(want, Object(l))
	}
	query := korrel8.Query(fmt.Sprintf("query_range?direction=FORWARD&query=%v", url.QueryEscape(`{test="loki"}`)))
	result := korrel8.NewListResult()
	require.NoError(t, s.Get(context.Background(), query, result))
	assert.Equal(t, want, result.List())
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
	s := NewStore(l.URL(), http.DefaultClient)
	require.NoError(t, err)

	for _, x := range []struct {
		query korrel8.Query
		want  []korrel8.Object
	}{
		{
			query: korrel8.Query(fmt.Sprintf("query_range?direction=FORWARD&query=%v&end=%v", url.QueryEscape(`{test="loki"}`), t1.UnixNano())),
			want:  []korrel8.Object{Object("much"), Object("too"), Object("early")},
		},
		{
			query: korrel8.Query(fmt.Sprintf("query_range?direction=FORWARD&query=%v&start=%v&end=%v", url.QueryEscape(`{test="loki"}`), t1.UnixNano(), t2.UnixNano())),
			want:  []korrel8.Object{Object("right"), Object("on"), Object("time")},
		},
	} {
		t.Run(string(x.query), func(t *testing.T) {
			var result korrel8.ListResult
			assert.NoError(t, s.Get(context.Background(), x.query, &result))
			assert.Equal(t, x.want, result.List())
		})
	}
}
