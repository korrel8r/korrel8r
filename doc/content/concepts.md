---
title: Concepts
weight: 3
---

# How Korrel8r Works

Korrel8r organizes observability data into **domains**, and uses **rules** to navigate between them.

## Domains organize data

A _domain_ represents one type of signal or resource. For example:

- `k8s` domain: Kubernetes resources (Pods, Services, Deployments, etc.)
- `log` domain: Application and system logs
- `metric` domain: Prometheus metrics
- `trace` domain: Distributed traces
- `alert` domain: Prometheus alerts

Available domains are described in the [Domain Reference](../reference/domains/).

Each domain defines the following abstractions, allowing Korrel8r to treat domains uniformly:

Object
: Individual data items returned by queries,
  for example a specific Pod, log entry, metric time series, trace span, etc.
  Signals and resources are all considered "objects".

Store
: A backend service that holds the data (Kubernetes API, Prometheus, Loki, etc.)

Class
: A specific type of data within a domain. Class names have the format `domain:class`.
  Examples: `k8s:Pod`, `k8s:Deployment`, `log:application`, `trace:span`

Query
: A request for data, formatted as `domain:class:selector`.
  The selector uses the native query language of the underlying store.
  Examples:
  - `k8s:Pod:{namespace: "default"}` (Kubernetes resource selector)
  - `log:application:{.kubernetes.namespace.name="default"}` (LogQL query)
  - `trace:span:{.k8s.namespace.name="foobar"}` (Tempo query)

## Rules connect data

_Rules_ express relationships between classes, possibly in different domains.
For example:
- "Pods belong to a Deployment" — relates `k8s:Pod` to `k8s:Deployment`
- "Pods generate logs" — relates `k8s:Pod` to `log:application`
- "Applications emit metrics" — relates `k8s:Pod` to `metric:metric`

Rules are [templates](../reference/configuration/#about-templates) that generate a _goal query_ using data from a _start object_.
The start and goal can be different classes, possibly from different domains.
If a rule cannot be applied to an object it may return a blank string, or raise an error.

The set of rules forms a graph connecting all the classes of data that korrel8r knows about.
Korrel8r works by traversing this graph: applying rules to some initial objects, executing the
resulting queries, retrieving more objects and so on.
