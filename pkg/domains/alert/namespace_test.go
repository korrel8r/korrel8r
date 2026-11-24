// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package alert

// This file contains unit tests for namespace extraction functionality.
// Tests the internal extractNamespacesFromQuery function used for non-admin user access control.

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/assert"
)

func TestExtractNamespacesFromQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    Query
		expected unique.Set[string]
	}{
		{
			name: "single namespace",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default"},
				},
			},
			expected: unique.NewSet("default"),
		},
		{
			name: "multiple different namespaces",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default"},
					{"namespace": "kube-system"},
				},
			},
			expected: unique.NewSet("default", "kube-system"),
		},
		{
			name: "duplicate namespaces",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default"},
					{"namespace": "default"},
				},
			},
			expected: unique.NewSet("default"),
		},
		{
			name: "no namespace",
			query: Query{
				Parsed: []map[string]string{
					{"alertname": "MyAlert"},
				},
			},
			expected: unique.Set[string]{},
		},
		{
			name: "empty namespace value",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": ""},
				},
			},
			expected: unique.Set[string]{},
		},
		{
			name: "mixed with and without namespace",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default", "alertname": "MyAlert"},
					{"alertname": "AnotherAlert"},
					{"namespace": "kube-system"},
				},
			},
			expected: unique.NewSet("default", "kube-system"),
		},
		{
			name: "namespace with special characters",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "openshift-monitoring"},
				},
			},
			expected: unique.NewSet("openshift-monitoring"),
		},
		{
			name: "empty query",
			query: Query{
				Parsed: []map[string]string{},
			},
			expected: unique.Set[string]{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNamespacesFromQuery(&tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
