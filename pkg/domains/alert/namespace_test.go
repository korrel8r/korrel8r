// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package alert

// This file contains unit tests for namespace extraction functionality.
// Tests the internal extractNamespacesFromQuery function used for non-admin user access control.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractNamespacesFromQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    Query
		expected map[string]bool
	}{
		{
			name: "single namespace",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default"},
				},
			},
			expected: map[string]bool{"default": true},
		},
		{
			name: "multiple different namespaces",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default"},
					{"namespace": "kube-system"},
				},
			},
			expected: map[string]bool{"default": true, "kube-system": true},
		},
		{
			name: "duplicate namespaces",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "default"},
					{"namespace": "default"},
				},
			},
			expected: map[string]bool{"default": true},
		},
		{
			name: "no namespace",
			query: Query{
				Parsed: []map[string]string{
					{"alertname": "MyAlert"},
				},
			},
			expected: map[string]bool{},
		},
		{
			name: "empty namespace value",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": ""},
				},
			},
			expected: map[string]bool{},
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
			expected: map[string]bool{"default": true, "kube-system": true},
		},
		{
			name: "namespace with special characters",
			query: Query{
				Parsed: []map[string]string{
					{"namespace": "openshift-monitoring"},
				},
			},
			expected: map[string]bool{"openshift-monitoring": true},
		},
		{
			name: "empty query",
			query: Query{
				Parsed: []map[string]string{},
			},
			expected: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNamespacesFromQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
