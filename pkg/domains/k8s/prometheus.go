// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"net"
	"net/url"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// TenancyPortQuery is the Thanos/Prometheus port for namespace-scoped query access.
	// Provides access to /api/v1/query, /api/v1/query_range, /api/v1/series endpoints.
	// Used by metric and incident domains.
	TenancyPortQuery = "9092"

	// TenancyPortRules is the Thanos/Prometheus port for namespace-scoped rules/alerts access.
	// Provides access to /api/v1/alerts and /api/v1/rules endpoints.
	// Used by alert domain.
	TenancyPortRules = "9093"
)

var prometheusLog = logging.Log()

// ReplacePort replaces the port in a host:port string.
// If the host doesn't have a port, it adds the new port.
func ReplacePort(host, newPort string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return net.JoinHostPort(h, newPort)
	}
	// If no port in host, add the new port
	return net.JoinHostPort(host, newPort)
}

// AddNamespaceParams adds namespace query parameters to the provided url.Values.
// This is required by prom-label-proxy on ports 9092 and 9093 for namespace scoping.
func AddNamespaceParams(params url.Values, namespaces map[string]bool) {
	for ns := range namespaces {
		params.Add("namespace", ns)
	}
}

// GetEffectivePrometheusURL returns a URL with the appropriate port based on user permissions.
// Admin users (with cluster-monitoring-view or cluster-monitoring-metrics-api) use the configured port (typically 9091).
// Non-admin users use the specified tenancy port for namespace-scoped access.
//
// Parameters:
//   - ctx: Context containing user authentication information
//   - baseURL: Original URL from configuration
//   - configuredPort: Port extracted from the configuration
//   - k8sClient: Kubernetes client for RBAC checks
//   - domain: Domain name for logging (e.g., "metric", "alert", "incident")
//   - tenancyPort: Port to use for non-admin users (e.g., TenancyPortQuery or TenancyPortRules)
//
// Returns the URL with the appropriate port set based on user permissions.
func GetEffectivePrometheusURL(ctx context.Context, baseURL *url.URL, configuredPort string, k8sClient client.Client, domain string, tenancyPort string) (*url.URL, error) {
	// Check if user has cluster-monitoring-view or cluster-monitoring-metrics-api permission
	hasClusterAccess, err := CanAccessPrometheusAPI(ctx, k8sClient)
	if err != nil {
		// On error, default to tenancy port (safer/more restrictive)
		prometheusLog.V(1).Info("failed to check Prometheus API permissions, using tenancy port", "error", err, "domain", domain)
		hasClusterAccess = false
	}

	// Clone the URL to avoid modifying the original
	u := &url.URL{}
	*u = *baseURL

	if !hasClusterAccess {
		// Non-admin user: change port to tenancy port for namespace-scoped access
		u.Host = ReplacePort(u.Host, tenancyPort)
		prometheusLog.V(2).Info("using tenancy port for query", "domain", domain, "port", tenancyPort)
	} else {
		// Admin user: keep the configured port (typically 9091)
		prometheusLog.V(2).Info("using configured port for query", "domain", domain, "port", configuredPort)
	}

	return u, nil
}
