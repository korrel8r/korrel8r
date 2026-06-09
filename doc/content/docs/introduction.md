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

## Correlation graphs

When korrel8r searches for correlated data, it returns a _correlation graph_.
The graph contains nodes, edges, queries, and counts — but not the full data objects themselves.

Each **node** represents a class of data (e.g. `k8s:Pod` or `log:application`).
Nodes contain:
- **queries** that will retrieve the actual data from the store.
- **counts** of how many items each query returns.
- **[statuses](../statuses/)** like `Error` or `Warning`, with counts.

**Edges** represent the correlation rules that connect one class to another.

### Graph-first workflow

This design lets you examine what data is available _before_ deciding what to retrieve.
For example, if one query returns 200 log records with 50 errors and another returns 1000 records with no errors,
you can check the 200 more interesting logs first — there is no need to retrieve the other 1000.

Following a chain of rules (e.g. Alert → Deployment → Pod → logs) requires retrieving intermediate data,
but the graph lets you skip intermediate steps and go straight to the results you need.

## Search strategies

Korrel8r offers two search strategies that traverse the rule graph in different ways.

### Goal search

Find paths from a starting point to one or more specific _goal_ classes.
Use this for targeted questions:
- "Find logs related to this pod"
- "What alerts fired for this deployment?"

Korrel8r finds the shortest paths through the rule graph from the start class to each goal class,
following rules and retrieving data along the way.

### Neighborhood search

Explore everything related to a starting point, up to a given _depth_ (number of rule hops).
Use this for open-ended investigation:
- "What is related to this pod?"
- "Show me everything connected to these alerts"

Korrel8r follows all rules reachable within the depth limit, building a graph of everything it finds.
