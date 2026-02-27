// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

// This file contains unit tests for namespace extraction functionality.
// Tests the internal extractNamespaces function used for non-admin user access control.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractNamespaces(t *testing.T) {
	tests := []struct {
		name      string
		selectors []string
		expected  map[string]bool
	}{
		{
			name:      "single namespace with double quotes",
			selectors: []string{`kube_pod_info{namespace="default"}`},
			expected:  map[string]bool{"default": true},
		},
		{
			name:      "single namespace with single quotes",
			selectors: []string{`kube_pod_info{namespace='default'}`},
			expected:  map[string]bool{"default": true},
		},
		{
			name: "multiple namespaces",
			selectors: []string{
				`kube_pod_info{namespace="default"}`,
				`kube_deployment_info{namespace="kube-system"}`,
			},
			expected: map[string]bool{"default": true, "kube-system": true},
		},
		{
			name: "duplicate namespaces",
			selectors: []string{
				`kube_pod_info{namespace="default"}`,
				`kube_deployment_info{namespace="default"}`,
			},
			expected: map[string]bool{"default": true},
		},
		{
			name:      "no namespace",
			selectors: []string{`kube_pod_info{app="myapp"}`},
			expected:  map[string]bool{},
		},
		{
			name:      "empty selectors",
			selectors: []string{},
			expected:  map[string]bool{},
		},
		{
			name: "mixed selectors with and without namespace",
			selectors: []string{
				`kube_pod_info{namespace="default"}`,
				`kube_deployment_info{app="myapp"}`,
			},
			expected: map[string]bool{"default": true},
		},
		{
			name: "namespace with special characters",
			selectors: []string{
				`kube_pod_info{namespace="openshift-monitoring"}`,
			},
			expected: map[string]bool{"openshift-monitoring": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNamespaces(tt.selectors)
			assert.Equal(t, tt.expected, result)
		})
	}
}
