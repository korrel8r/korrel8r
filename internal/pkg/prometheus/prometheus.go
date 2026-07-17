// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package prometheus

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/cache"
	"github.com/korrel8r/korrel8r/pkg/api/auth"
	authv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NamespacedPort is the Thanos/Prometheus port for namespace-scoped query access.
	// Provides access to /api/v1/query, /api/v1/query_range, /api/v1/series endpoints.
	NamespacedPort = "9092"
)

// ReplacePort replaces the port in a host:port string.
// If the host doesn't have a port, it adds the new port.
func ReplacePort(host, newPort string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return net.JoinHostPort(h, newPort)
	}
	return net.JoinHostPort(host, newPort)
}

// AddNamespaceParams adds namespace query parameters to the provided url.Values.
// This is required by prom-label-proxy on ports 9092 and 9093 for namespace scoping.
func AddNamespaceParams(params url.Values, namespaces map[string]bool) {
	for ns := range namespaces {
		params.Add("namespace", ns)
	}
}

// EffectiveURL returns a clone of baseURL with the namespaced port substituted
// when the user lacks cluster-level Prometheus access. The original URL is never modified.
func EffectiveURL(ctx context.Context, baseURL *url.URL, k8sClient client.Client) *url.URL {
	hasClusterAccess, _ := CanAccessPrometheusAPI(ctx, k8sClient)
	u := *baseURL
	if !hasClusterAccess {
		u.Host = ReplacePort(u.Host, NamespacedPort)
	}
	return &u
}

var rbacCache = cache.NewTTL[string, bool](10 * time.Minute)

// CanAccessPrometheusAPI checks if the user in the context has permission to access
// the full Prometheus API (port 9091). Results are cached per bearer token.
func CanAccessPrometheusAPI(ctx context.Context, c client.Client) (bool, error) {
	token := auth.ContextToken(ctx)
	if allowed, ok := rbacCache.Get(token); ok {
		return allowed, nil
	}

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

	rbacCache.Put(token, sar.Status.Allowed)
	return sar.Status.Allowed, nil
}
