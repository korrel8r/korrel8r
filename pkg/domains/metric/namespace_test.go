// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackError(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"http error", "unavailable", http.StatusServiceUnavailable},
		{"unexpected status", `{"status":"error"}`, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			store, err := NewStore(server.URL, "", server.Client())
			require.NoError(t, err)
			err = store.Get(context.Background(), Query(`up?namespace=ns-a`), nil, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, "used main metric URL, namespaced URL not configured")
		})
	}
}

func TestQueryTarget(t *testing.T) {
	baseURL, err := url.Parse("https://prometheus.example.com:9091/base")
	require.NoError(t, err)
	namespacedURL, err := url.Parse("https://namespace-scoped.example.com/custom")
	require.NoError(t, err)

	store := &Store{baseURL: baseURL, namespacedURL: namespacedURL}
	t.Run("explicit suffix", func(t *testing.T) {
		params := url.Values{"namespace": {"ns-a", "ns-b"}}
		got, namespaces, ignoredNamespaces := store.queryTarget(params)
		assert.False(t, ignoredNamespaces)
		assert.Equal(t, namespacedURL, got)
		assert.Equal(t, unique.NewSet("ns-a", "ns-b"), namespaces)
		assert.Equal(t, "prometheus.example.com:9091", baseURL.Host)
	})

	t.Run("no suffix", func(t *testing.T) {
		got, namespaces, ignoredNamespaces := store.queryTarget(nil)
		assert.False(t, ignoredNamespaces)
		assert.Equal(t, baseURL, got)
		assert.Empty(t, namespaces)
	})

	t.Run("explicit suffix without namespaced URL", func(t *testing.T) {
		params := url.Values{"namespace": {"ns-a"}}
		got, namespaces, ignoredNamespaces := (&Store{baseURL: baseURL}).queryTarget(params)
		assert.True(t, ignoredNamespaces)
		assert.Equal(t, baseURL, got)
		assert.Equal(t, unique.NewSet("ns-a"), namespaces)
	})
}
