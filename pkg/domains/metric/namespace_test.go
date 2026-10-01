// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryTarget(t *testing.T) {
	baseURL, err := url.Parse("https://prometheus.example.com:9091/base")
	require.NoError(t, err)
	namespacedURL, err := url.Parse("https://namespace-scoped.example.com/custom")
	require.NoError(t, err)

	store := &Store{baseURL: baseURL, namespacedURL: namespacedURL}
	t.Run("explicit suffix", func(t *testing.T) {
		params := url.Values{"namespace": {"ns-a", "ns-b"}}
		got, namespaces := store.queryTarget(params)
		assert.Equal(t, namespacedURL, got)
		assert.Equal(t, map[string]bool{"ns-a": true, "ns-b": true}, namespaces)
		assert.Equal(t, "prometheus.example.com:9091", baseURL.Host)
	})

	t.Run("no suffix", func(t *testing.T) {
		got, namespaces := store.queryTarget(nil)
		assert.Equal(t, baseURL, got)
		assert.Empty(t, namespaces)
	})

	t.Run("explicit suffix without namespaced URL", func(t *testing.T) {
		params := url.Values{"namespace": {"ns-a"}}
		got, namespaces := (&Store{baseURL: baseURL}).queryTarget(params)
		assert.Equal(t, baseURL, got)
		assert.Equal(t, map[string]bool{"ns-a": true}, namespaces)
	})
}
