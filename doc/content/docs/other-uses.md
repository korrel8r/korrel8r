---
title: Other Uses
description: Scripting, automation, and custom integrations
weight: 50
---

Korrel8r is most commonly used for troubleshooting via the
[OpenShift console](../troubleshooting-panel/) or [AI agents](../ai-agents/),
but its correlation engine, data access, and APIs make it useful in other contexts too.

## Uniform Data Access

Korrel8r can act as a single client for all the observability stores it connects to.
Instead of using separate tools for each backend --
`kubectl` for Kubernetes, `logcli` for Loki, `promcli` for Prometheus --
you can use korrel8r with one consistent query format.

```bash
# Kubernetes resources
korrel8r objects --query 'k8s:Pod:{namespace: myapp}'

# Logs from Loki
korrel8r objects --query 'log:application:{kubernetes_namespace_name="myapp"}'

# Prometheus metrics
korrel8r objects --query 'metric:metric:{namespace="myapp"}'
```

Each query uses the native query language of the underlying store (label selectors, LogQL, PromQL),
prefixed with `domain:class:` to route it to the right backend.
See the [Domain Reference](../reference/domains/) for the query syntax of each domain.

As a service, korrel8r provides token-forwarding so clients authenticate once
and korrel8r handles per-store authentication on their behalf.

## Scripting and Automation

The [REST API](../reference/rest/) and [command line](../reference/cmd/) make it easy to
integrate korrel8r into scripts and automation workflows.

For example, an automated incident response script might:

1. Receive an alert notification.
2. Call korrel8r to find all signals correlated with the alert.
3. Collect the relevant logs, metrics, and resource state.
4. Assemble a diagnostic report.

```bash
# Find all data related to a firing alert
korrel8r neighbors --query 'alert:alert:{alertname="HighErrorRate"}' --depth 3
```

## Custom Integrations

Korrel8r's [REST API](../reference/rest/) serves as a building block for custom tools:

- **Dashboards**: build correlation-aware views that pull data from multiple stores.
- **ChatOps**: connect korrel8r to Slack or other messaging tools for on-demand correlation queries.
- **CI/CD pipelines**: automatically gather correlated signals when a deployment fails.

The [MCP interface](../ai-agents/) makes korrel8r accessible to any MCP-compatible tool,
not just AI agents.

## Extending Korrel8r

You can extend korrel8r in two ways without modifying the code:

- **Add rules**: define new correlations between existing domains.
  See [Writing Rules](../writing-rules/).
- **Add stores**: configure additional store connections for existing domains.
  See [Configuration](../configuration/#stores).

Adding entirely new domains (new signal types, query languages, or data stores)
requires Go code changes.
See the [developer guide](https://github.com/korrel8r/korrel8r/blob/main/CLAUDE.md) and
existing domain implementations in `pkg/domains/` for patterns to follow.
