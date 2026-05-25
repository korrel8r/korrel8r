---
title: Korrel8r
weight: 1
---

# Korrel8r

*Navigate relationships between cluster resources and observability signals.*

Observability for Kubernetes clusters often involves multiple systems:
logs in Loki, metrics in Prometheus, traces in Tempo, events in the API server,
alerts in Alertmanager and so on.
Each system has different data models, query languages, and APIs.
When troubleshooting, you need to piece together information from multiple sources.

Korrel8r is a _rule-based correlation engine_ that automatically discovers and graphs
related resources and observability signals across multiple data stores.
Given any starting point — an alert, a service or deployment that is in trouble etc —
korrel8r will search for related data, potentially following multiple relationships
between objects in different data stores.

Goal Search
: Find paths from starting objects to a specific type of target data.
  For example: "Find all logs related to this Alert" might cause korrel8r to connect
  the alert to related deployments, the deployments to their pods, and finally the pods to logs.

Neighborhood Search
: Find all data reachable within N steps from starting objects.
  For example: "Show me everything related to this Pod within 2 steps" might return
  deployments and services that own the Pod, logs from the pods containers,
  metrics describing the Pod, network flows with the Pod as an endpoint and so on.

## Key Capabilities

- **Universal correlation**: Connects data across multiple formats, stores, and query languages (OTEL, Prometheus/PromQL, Loki/LogQL, etc.)
- **Multiple stores**: Relationships can span data in Prometheus, Loki, Alertmanager, Kubernetes API server, and more
- **Extensible rules**: Configurable YAML rules define how different data types relate to each other
- **Extensible domains**: Add new domains to handle new signal types, query languages, and data stores
- **Flexible deployment**: Deploy as a REST service or use from the command line

## Who Uses Korrel8r

- **Cluster administrators, SREs, and developers** who need to troubleshoot issues across complex Kubernetes environments
- **Observability tool builders** who want to display and manipulate correlation graphs
- **Integrated solutions** — for example the [OpenShift Console Troubleshooting Panel](https://github.com/openshift/troubleshooting-panel-console-plugin)

## Relationship to OpenTelemetry

The [OpenTelemetry](https://opentelemetry.io) project (OTEL) defines standard vocabularies for traces, metrics, and logs.
Korrel8r can correlate OTEL data with other data formats for observability signals.

Korrel8r complements OTEL by:

- Bridging between OTEL and non-OTEL systems
- Providing correlation rules between different signal types
- Supporting data types beyond the current OTEL specification (like Kubernetes events or network flows)
- Working with any query language or storage system

# Documentation

- [Getting Started](getting-started/) — Installation, running, and first queries
- [Concepts](concepts/) — How domains, classes, queries, and rules work

## Reference

- [Korrel8r Command](reference/cmd/) — Command-line reference
- [Domains](reference/domains/) — Signal domains, classes, and query syntax
- [Client Command](https://korrel8r.github.io/client/) — Command-line REST client for korrel8r servers
- [Configuration](reference/configuration/) — Configuration file format, stores, rules, and templates
- [REST API](reference/rest/) — HTTP API reference

## Links

- [GitHub Repository](https://github.com/korrel8r/korrel8r)
