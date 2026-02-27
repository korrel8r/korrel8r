// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CanAccessPrometheusAPI checks if the user in the context has permission to access
// the full Prometheus API (port 9091). This requires either:
// - cluster-monitoring-view cluster role, or
// - cluster-monitoring-metrics-api cluster role
//
// The check is performed using SubjectAccessReview to verify the actual permission
// rather than checking for specific role names, which is more flexible.
//
// Returns true if the user has permission, false otherwise.
func CanAccessPrometheusAPI(ctx context.Context, c client.Client) (bool, error) {
	// Check if user can access prometheuses.monitoring.coreos.com/api resource
	// This permission is granted by cluster-monitoring-view or cluster-monitoring-metrics-api roles
	sar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Group:       "monitoring.coreos.com",
				Resource:    "prometheuses",
				Subresource: "api",
				Name:        "k8s",
				Namespace:   "openshift-monitoring",
				Verb:        "get",
			},
		},
	}

	if err := c.Create(ctx, sar); err != nil {
		return false, err
	}

	return sar.Status.Allowed, nil
}
