---
title: Introduction
description: How domains, classes, queries, and rules work
weight: 2
---

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
: An individual data item — a Pod, a log entry, a metric time series, a trace span.
  Signals and resources are all "objects" to korrel8r.

Store
: A backend that holds objects — Kubernetes API, Prometheus, Loki, Tempo, etc.

Class
: A type of object within a domain, written as `domain:class`.
  Examples: `k8s:Pod`, `k8s:Deployment.apps`, `log:application`, `metric:metric`

Query
: A request for objects of a class, written as `domain:class:selector`.
  The selector uses the native query language of the store.
  Examples:
  - `k8s:Pod:{namespace: "default"}` — Kubernetes label/field selector
  - `log:application:{kubernetes_namespace_name="default"}` — LogQL
  - `metric:metric:{namespace="default"}` — PromQL

For the full list of domains and their query syntax, see the [Domain Reference](../reference/domains/).

## Rules connect data

_Rules_ express relationships between start and goal classes, which can be in different domains.
For example:
- "Pods belong to a Deployment" — relates `k8s:Pod` to `k8s:Deployment`
- "Pods generate logs" — relates `k8s:Pod` to `log:application`
- "Applications emit metrics" — relates `k8s:Pod` to `metric:metric`

Rules are applied to an object of the start class, and generate queries for the goal class.
The set of rules forms a graph connecting all the classes of data that korrel8r knows about.
Korrel8r works by traversing this graph: applying rules to some initial objects, executing the
resulting queries to retrieve more objects, applying more rules, and so on.

Korrel8r comes with a comprehensive set of rules for Kubernetes and observability data.
You can also [write your own rules](../writing-rules/) to handle custom relationships.
