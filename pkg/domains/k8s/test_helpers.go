// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// interceptorFuncs creates interceptor functions that simulate SelfSubjectAccessReview responses
// This is a test helper function used by multiple test files
func interceptorFuncs(allowed bool) interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if sar, ok := obj.(*authv1.SelfSubjectAccessReview); ok {
				// Simulate the response
				sar.Status.Allowed = allowed
			}
			return nil
		},
	}
}

// interceptorFuncsCapture creates interceptor functions that capture and simulate SelfSubjectAccessReview responses
// This is a test helper function used by RBAC tests
func interceptorFuncsCapture(allowed bool, captureSAR *authv1.SelfSubjectAccessReview) interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if sar, ok := obj.(*authv1.SelfSubjectAccessReview); ok {
				// Capture the SAR if requested
				if captureSAR != nil {
					*captureSAR = *sar
				}
				// Simulate the response
				sar.Status.Allowed = allowed
			}
			return nil
		},
	}
}
