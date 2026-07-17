// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package prometheus

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/api/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
			assert.ElementsMatch(t, tt.expected, params["namespace"])
		})
	}
}

func TestEffectiveURL(t *testing.T) {
	rbacCache.Clear()
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	baseURL, err := url.Parse("https://prometheus.example.com:9091/path")
	require.NoError(t, err)

	t.Run("admin uses configured port", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptorFuncs(true)).Build()
		ctx := auth.WithToken(context.Background(), "eff-admin")
		u := EffectiveURL(ctx, baseURL, c)
		assert.Equal(t, "9091", u.Port())
		assert.Equal(t, "/path", u.Path)
	})

	t.Run("non-admin uses namespaced port", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptorFuncs(false)).Build()
		ctx := auth.WithToken(context.Background(), "eff-nonadmin")
		u := EffectiveURL(ctx, baseURL, c)
		assert.Equal(t, NamespacedPort, u.Port())
		assert.Equal(t, "/path", u.Path)
	})

	t.Run("does not modify original URL", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptorFuncs(false)).Build()
		ctx := auth.WithToken(context.Background(), "eff-preserve")
		_ = EffectiveURL(ctx, baseURL, c)
		assert.Equal(t, "9091", baseURL.Port())
	})
}

func TestCanAccessPrometheusAPI(t *testing.T) {
	rbacCache.Clear()
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	tests := []struct {
		name           string
		allowed        bool
		expectedResult bool
		expectedErr    bool
	}{
		{
			name:           "admin user with cluster-monitoring-view permission",
			allowed:        true,
			expectedResult: true,
		},
		{
			name:           "non-admin user without cluster-monitoring-view permission",
			allowed:        false,
			expectedResult: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptorFuncs(tt.allowed)).
				Build()

			ctx := auth.WithToken(context.Background(), fmt.Sprintf("test-token-%d", i))
			result, err := CanAccessPrometheusAPI(ctx, fakeClient)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestCanAccessPrometheusAPI_VerifiesCorrectResource(t *testing.T) {
	rbacCache.Clear()
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	var capturedSAR authv1.SelfSubjectAccessReview

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptorFuncsCapture(true, &capturedSAR)).
		Build()

	ctx := auth.WithToken(context.Background(), "test-token-verify")
	_, err := CanAccessPrometheusAPI(ctx, fakeClient)
	require.NoError(t, err)

	assert.NotNil(t, capturedSAR.Spec.ResourceAttributes)
	assert.Equal(t, "monitoring.coreos.com", capturedSAR.Spec.ResourceAttributes.Group)
	assert.Equal(t, "prometheuses", capturedSAR.Spec.ResourceAttributes.Resource)
	assert.Equal(t, "api", capturedSAR.Spec.ResourceAttributes.Subresource)
	assert.Equal(t, "k8s", capturedSAR.Spec.ResourceAttributes.Name)
	assert.Equal(t, "openshift-monitoring", capturedSAR.Spec.ResourceAttributes.Namespace)
	assert.Equal(t, "get", capturedSAR.Spec.ResourceAttributes.Verb)
}

func interceptorFuncs(allowed bool) interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if sar, ok := obj.(*authv1.SelfSubjectAccessReview); ok {
				sar.Status.Allowed = allowed
			}
			return nil
		},
	}
}

func interceptorFuncsCapture(allowed bool, captureSAR *authv1.SelfSubjectAccessReview) interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if sar, ok := obj.(*authv1.SelfSubjectAccessReview); ok {
				if captureSAR != nil {
					*captureSAR = *sar
				}
				sar.Status.Allowed = allowed
			}
			return nil
		},
	}
}
