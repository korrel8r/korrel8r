// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCanAccessPrometheusAPI(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	tests := []struct {
		name           string
		allowed        bool
		createErr      bool
		expectedResult bool
		expectedErr    bool
	}{
		{
			name:           "admin user with cluster-monitoring-view permission",
			allowed:        true,
			createErr:      false,
			expectedResult: true,
			expectedErr:    false,
		},
		{
			name:           "non-admin user without cluster-monitoring-view permission",
			allowed:        false,
			createErr:      false,
			expectedResult: false,
			expectedErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with interceptor to simulate SelfSubjectAccessReview response
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithInterceptorFuncs(interceptorFuncs(tt.allowed)).
				Build()

			ctx := context.Background()
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
	scheme := runtime.NewScheme()
	require.NoError(t, authv1.AddToScheme(scheme))

	var capturedSAR authv1.SelfSubjectAccessReview

	// Create fake client with interceptor to capture the SAR request
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptorFuncsCapture(true, &capturedSAR)).
		Build()

	ctx := context.Background()
	_, err := CanAccessPrometheusAPI(ctx, fakeClient)
	require.NoError(t, err)

	// Verify the correct resource is being checked
	assert.NotNil(t, capturedSAR.Spec.ResourceAttributes)
	assert.Equal(t, "monitoring.coreos.com", capturedSAR.Spec.ResourceAttributes.Group)
	assert.Equal(t, "prometheuses", capturedSAR.Spec.ResourceAttributes.Resource)
	assert.Equal(t, "api", capturedSAR.Spec.ResourceAttributes.Subresource)
	assert.Equal(t, "k8s", capturedSAR.Spec.ResourceAttributes.Name)
	assert.Equal(t, "openshift-monitoring", capturedSAR.Spec.ResourceAttributes.Namespace)
	assert.Equal(t, "get", capturedSAR.Spec.ResourceAttributes.Verb)
}
