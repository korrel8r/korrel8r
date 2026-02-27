// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReplacePort(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		newPort  string
		expected string
	}{
		{
			name:     "replace existing port",
			host:     "prometheus.example.com:9091",
			newPort:  "9092",
			expected: "prometheus.example.com:9092",
		},
		{
			name:     "add port when missing",
			host:     "prometheus.example.com",
			newPort:  "9092",
			expected: "prometheus.example.com:9092",
		},
		{
			name:     "replace port in localhost",
			host:     "localhost:9091",
			newPort:  "9093",
			expected: "localhost:9093",
		},
		{
			name:     "replace port in IP address",
			host:     "10.0.0.1:9091",
			newPort:  "9092",
			expected: "10.0.0.1:9092",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplacePort(tt.host, tt.newPort)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddNamespaceParams(t *testing.T) {
	tests := []struct {
		name       string
		namespaces map[string]bool
		expected   []string
	}{
		{
			name:       "single namespace",
			namespaces: map[string]bool{"default": true},
			expected:   []string{"default"},
		},
		{
			name:       "multiple namespaces",
			namespaces: map[string]bool{"ns1": true, "ns2": true, "ns3": true},
			expected:   []string{"ns1", "ns2", "ns3"},
		},
		{
			name:       "empty namespaces",
			namespaces: map[string]bool{},
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			AddNamespaceParams(params, tt.namespaces)

			actualNamespaces := params["namespace"]
			assert.ElementsMatch(t, tt.expected, actualNamespaces)
		})
	}
}

func TestGetEffectivePrometheusURL_AdminUser(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	// Create fake client that simulates admin user (allowed=true)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptorFuncs(true)).
		Build()

	baseURL, err := url.Parse("https://prometheus.example.com:9091/")
	require.NoError(t, err)

	tests := []struct {
		name         string
		domain       string
		tenancyPort  string
		expectedPort string
	}{
		{
			name:         "admin user for metrics uses configured port",
			domain:       "metric",
			tenancyPort:  TenancyPortQuery,
			expectedPort: "9091",
		},
		{
			name:         "admin user for alerts uses configured port",
			domain:       "alert",
			tenancyPort:  TenancyPortRules,
			expectedPort: "9091",
		},
		{
			name:         "admin user for incidents uses configured port",
			domain:       "incident",
			tenancyPort:  TenancyPortQuery,
			expectedPort: "9091",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			effectiveURL, err := GetEffectivePrometheusURL(ctx, baseURL, "9091", fakeClient, tt.domain, tt.tenancyPort)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPort, effectiveURL.Port())
			assert.Equal(t, "prometheus.example.com", effectiveURL.Hostname())
			assert.Equal(t, "https", effectiveURL.Scheme)
		})
	}
}

func TestGetEffectivePrometheusURL_NonAdminUser(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	// Create fake client that simulates non-admin user (allowed=false)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptorFuncs(false)).
		Build()

	baseURL, err := url.Parse("https://prometheus.example.com:9091/")
	require.NoError(t, err)

	tests := []struct {
		name         string
		domain       string
		tenancyPort  string
		expectedPort string
	}{
		{
			name:         "non-admin user for metrics uses tenancy port 9092",
			domain:       "metric",
			tenancyPort:  TenancyPortQuery,
			expectedPort: "9092",
		},
		{
			name:         "non-admin user for alerts uses tenancy port 9093",
			domain:       "alert",
			tenancyPort:  TenancyPortRules,
			expectedPort: "9093",
		},
		{
			name:         "non-admin user for incidents uses tenancy port 9092",
			domain:       "incident",
			tenancyPort:  TenancyPortQuery,
			expectedPort: "9092",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			effectiveURL, err := GetEffectivePrometheusURL(ctx, baseURL, "9091", fakeClient, tt.domain, tt.tenancyPort)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPort, effectiveURL.Port())
			assert.Equal(t, "prometheus.example.com", effectiveURL.Hostname())
			assert.Equal(t, "https", effectiveURL.Scheme)
		})
	}
}

func TestGetEffectivePrometheusURL_PermissionCheckError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	// Create fake client that returns an error (but we still expect fallback to tenancy port)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	baseURL, err := url.Parse("https://prometheus.example.com:9091/")
	require.NoError(t, err)

	ctx := context.Background()
	effectiveURL, err := GetEffectivePrometheusURL(ctx, baseURL, "9091", fakeClient, "metric", TenancyPortQuery)

	// Should not error, but should fall back to tenancy port
	require.NoError(t, err)
	assert.Equal(t, TenancyPortQuery, effectiveURL.Port(), "should use tenancy port on permission check error")
}

func TestGetEffectivePrometheusURL_PreservesOriginalURL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptorFuncs(false)).
		Build()

	baseURL, err := url.Parse("https://prometheus.example.com:9091/path/to/api")
	require.NoError(t, err)

	ctx := context.Background()
	effectiveURL, err := GetEffectivePrometheusURL(ctx, baseURL, "9091", fakeClient, "metric", TenancyPortQuery)

	require.NoError(t, err)
	assert.Equal(t, TenancyPortQuery, effectiveURL.Port())
	assert.Equal(t, "/path/to/api", effectiveURL.Path, "should preserve URL path")
	assert.Equal(t, baseURL.String(), "https://prometheus.example.com:9091/path/to/api", "original URL should be unmodified")
}
