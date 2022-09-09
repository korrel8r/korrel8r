package loki

import (
	"context"
	"net/http"
	"testing"

	"github.com/alanconway/korrel8/pkg/internal/test"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Query(t *testing.T) {
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
	assert.Equal(t, want, got)
}
