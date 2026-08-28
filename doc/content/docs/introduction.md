---
title: Introduction
description: How domains, classes, queries, and rules work
weight: 2
---

Korrel8r is a rule-based correlation engine for Kubernetes clusters.
Given any starting point — an alert, a pod, a deployment — it finds related data
across multiple signal stores (Prometheus, Loki, Tempo, Kubernetes API)
by searching with different APIs and query languages automatically.

## Domains

A _domain_ represents one type of signal or resource.
Available domains are described in the [Domain Reference](../reference/domains/).

| Domain | Signal | Store |
|--------|--------|-------|
| `k8s` | Kubernetes resources | Kubernetes API |
| `log` | Application and system logs | Loki |
| `metric` | Prometheus metrics | Prometheus |
| `alert` | Prometheus alerts | Alertmanager |
| `trace` | Distributed traces | Tempo |
| `netflow` | Network flows | Loki |

Each domain defines four abstractions:

Object
: An individual data item — a Pod, a log entry, a metric time series, a trace span.

Store
: A backend that holds objects.

Class
: A type of object, written as `domain:class`.
  E.g. `k8s:Pod`, `log:application`, `metric:metric`.

Query
: A request for objects, written as `domain:class:selector` using the store's native query language.
  E.g. `k8s:Pod:{namespace: "default"}`, `log:application:{kubernetes_namespace_name="default"}`.

## Rules

_Rules_ express relationships between a start class and a goal class, which can be in different domains.
A rule is applied to an object of the start class and generates a query for the goal class.
For example:
- `k8s:Pod` → `k8s:Deployment` (pods belong to a deployment)
- `k8s:Pod` → `log:application` (pods generate logs)
- `k8s:Pod` → `metric:metric` (pods emit metrics)

The set of rules forms a graph connecting all classes.
Korrel8r traverses this graph: apply rules to objects, execute the resulting queries, apply more rules, and repeat.

Korrel8r ships with rules for Kubernetes and observability data.
You can [write your own](../writing-rules/).

## Correlation graphs

Searches return a _correlation graph_ — not the data itself, but a map of what's available.

Each **node** is a class of data, containing:
- **queries** to retrieve the actual objects from the store
- **counts** of matching items
- **[statuses](../statuses/)** like `Error` or `Warning`, with counts

Each **edge** is a rule connecting one class to another.

This lets you examine what data exists _before_ deciding what to retrieve.
For example, if one node shows 200 logs with 50 errors and another shows 1000 logs with none,
you can go straight to the interesting ones.

## Search strategies

### Goal search

Find shortest paths from a start to specific goal classes.
Use for targeted questions: _"find logs for this pod"_, _"what alerts fired for this deployment?"_

### Neighborhood search

Explore everything reachable within N rule hops from a start.
Use for open-ended investigation: _"what is related to this pod?"_
