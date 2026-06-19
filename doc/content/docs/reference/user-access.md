---
title: Admin vs Regular User
description: Understanding how admin and non-admin users access observability data
weight: 40
---

Korrel8r respects Kubernetes RBAC permissions when accessing observability data. This guide explains the differences between admin and non-admin user access patterns.

## Access Levels

Korrel8r automatically detects user permissions using Kubernetes SubjectAccessReview and selects the appropriate backend ports and APIs. No configuration changes are needed.

### Admin Users

Admin users have **cluster-wide** access to all observability signals.

**Required RBAC:**
```bash
# Grant cluster-wide monitoring access (recommended)
oc adm policy add-cluster-role-to-user cluster-monitoring-view <username>

# Alternative: metrics-only access
oc adm policy add-cluster-role-to-user cluster-monitoring-metrics-api <username>
```

**Capabilities:**
- ✅ Query data from all namespaces
- ✅ Access cluster-scoped resources (Nodes, PersistentVolumes, etc.)
- ✅ Optional namespace filtering (can query with or without namespace)
- ✅ Direct access to backend services (ports 9091, 9094)

### Non-Admin Users

Non-admin users have **namespace-scoped** access limited to their assigned namespaces.

**Required RBAC:**
```bash
# Minimum: view access to namespace
oc adm policy add-role-to-user view <username> -n <namespace>

# For alerts: monitoring access
oc adm policy add-role-to-user monitoring-edit <username> -n <namespace>
```

**Capabilities:**
- ✅ Query data from assigned namespaces only
- ✅ Access namespace-scoped resources
- ❌ Cannot access cluster-scoped resources (Nodes)
- ⚠️ Must include namespace in queries for some data types
- ⚠️ Access via tenancy layer (ports 9092, 9093)

## Port Mapping

OpenShift monitoring services expose different ports for different access levels:

| Service | Port | Access Level | Required RBAC |
|---------|------|--------------|---------------|
| **Prometheus/Thanos** | 9091 | Cluster-wide | `cluster-monitoring-view` |
| **Prometheus/Thanos** | 9092 | Namespace-scoped (query) | `view` in namespace |
| **Prometheus/Thanos** | 9093 | Namespace-scoped (rules) | `monitoring-edit` in namespace |
| **Alertmanager** | 9094 | Cluster-wide | `cluster-monitoring-view` |
| **Alertmanager** | 9092 | Namespace-scoped | `monitoring-edit` in namespace |

Korrel8r automatically selects the correct port based on user permissions.

## Alerts

Korrel8r supports two types of alerts with different access patterns:

- **Prometheus alerts** — Platform alerts from PrometheusRule CRDs
- **Loki alerts** — Log-based alerts from AlertingRule CRDs

### How Alerts Work Differently

**Prometheus Alerts:**
- Single API call to Prometheus Rules endpoint
- Returns both rule definitions and firing instances
- Works consistently for all user types

**Loki Alerts:**
- Requires two API calls:
  1. Loki Ruler API → Rule definitions
  2. Alertmanager API → Firing instances
- Alertmanager tenancy port (9092) requires namespace parameter
- Additional requirements for non-admin users

### Label Format Differences

Alerts use different label naming conventions depending on source:

| Resource Field | Prometheus Labels | Loki/OTEL Labels |
|----------------|-------------------|------------------|
| Pod name | `pod` | `kubernetes_pod_name` |
| Namespace | `namespace` | `kubernetes_namespace_name` |
| Container | `container` | `kubernetes_container_name` |

Korrel8r correlation rules handle both formats automatically.

### Admin Alert Access

Admin users can query alerts without namespace restrictions:

```bash
# Both Prometheus and Loki alerts work without namespace
korrel8rcli --query 'alert:alert:{"alertname":"PodMemoryUsageHigh"}'
korrel8rcli --query 'alert:alert:{"alertname":"DevAppLogVolumesHigh"}'

# Can optionally filter by namespace
korrel8rcli --query 'alert:alert:{"alertname":"MyAlert","namespace":"my-ns"}'
```

**Ports Used:**
- Alertmanager: **9094** (cluster-wide)
- Prometheus: **9091** (cluster-wide)

### Non-Admin Alert Access

Non-admin users **must include namespace** when querying Loki alerts:

```bash
# ❌ FAILS for non-admin (Loki alerts)
korrel8rcli --query 'alert:alert:{"alertname":"DevAppLogVolumesHigh"}'

# ✅ WORKS for non-admin (Loki alerts)
korrel8rcli --query 'alert:alert:{"alertname":"DevAppLogVolumesHigh","namespace":"test-log-alerts"}'

# Prometheus alerts work with namespace filtering
korrel8rcli --query 'alert:alert:{"alertname":"PodMemoryUsageHigh","namespace":"my-ns"}'
```

**Why Namespace is Required:**

The Alertmanager tenancy port (9092) requires a `?namespace=` query parameter. Korrel8r automatically injects this parameter when namespace is present in the alert query.

**Ports Used:**
- Alertmanager: **9092** (namespace-scoped, requires namespace parameter)
- Prometheus: **9093** (namespace-scoped)

### Alert Access Summary

| Aspect | Prometheus Alerts | Loki Alerts (Admin) | Loki Alerts (Non-Admin) |
|--------|-------------------|---------------------|-------------------------|
| **API Calls** | Single (Prometheus) | Two (Loki + Alertmanager) | Two (Loki + Alertmanager) |
| **Port** | 9091/9094 | 9091/9094 | 9093/9092 |
| **Namespace Required** | Optional | Optional | **Required** |
| **Label Format** | Standard (`pod`, `namespace`) | OTEL (`kubernetes_pod_name`) | OTEL (`kubernetes_pod_name`) |
| **RBAC** | `cluster-monitoring-view` | `cluster-monitoring-view` | `monitoring-edit` in namespace |

### Console UI Limitation

The OpenShift Console troubleshooting panel shows alerts in the correlation graph, but the alert detail page may not display Loki alerts correctly.

**Current Status:**
- ✅ Loki alerts appear in troubleshooting panel graph
- ✅ CLI (`korrel8rcli`) works for both admin and non-admin
- ✅ REST API works for both admin and non-admin
- ⚠️ Console Alerting page may not recognize Loki alert label format

This is a Console UI limitation—the alert detail page filters using standard Prometheus label names and doesn't recognize OTEL/Viaq formats used by Loki alerts.

## Troubleshooting

### Checking Your Permissions

Verify your current access level:

```bash
# Check if you have cluster-wide monitoring access
oc auth can-i get prometheuses.monitoring.coreos.com/api --subresource=api -n openshift-monitoring

# Returns "yes" for admin, "no" for non-admin
```

### No Results for Non-Admin User

**Symptom:** Query returns `{"edges":null,"nodes":null}` or empty results

**Solution:** Include namespace in your query

```bash
# Before (fails for non-admin Loki alerts)
alert:alert:{"alertname":"MyAlert"}

# After (works for non-admin)
alert:alert:{"alertname":"MyAlert","namespace":"my-namespace"}
```

### Permission Denied Errors

**Symptom:** HTTP 403 Forbidden or "cannot access prometheuses.monitoring.coreos.com/api"

**Solution:** Grant appropriate RBAC permissions

```bash
# For admin access
oc adm policy add-cluster-role-to-user cluster-monitoring-view <username>

# For namespace access
oc adm policy add-role-to-user monitoring-edit <username> -n <namespace>
```

### Viewing Korrel8r Port Selection

Enable verbose logging to see which ports Korrel8r selects:

```bash
# Using korrel8rcli
korrel8rcli config --set-verbose=3

# Using curl
curl --oauth2-bearer $(oc whoami -t) -X PUT $KORREL8R_URL/api/v1alpha1/config?verbose=3
```

Check logs for port selection messages:
```bash
oc logs -n korrel8r deployment/korrel8r | grep "using.*port"
```

Expected output:
- Admin: `"using configured port" port="9091"`
- Non-admin: `"using tenancy port" port="9092"`

## Additional Resources

- [Alert Domain Reference](../domains/alert/) — Alert query syntax and configuration
- [Configuration Reference](../configuration/) — Store and rule configuration
- [REST API Reference](../rest/) — API endpoints and parameters
