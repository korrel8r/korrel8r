---
title: Korrel8r
weight: 1
description: Korrel8r Site
---
<br>

>Correlate observability signals and Kubernetes resources.

Korrel8r is a _rule-based correlation engine_ for Kubernetes clusters.
Given any starting point — an alert, a pod, a deployment —
it follows configurable rules to find related data across stores like
Prometheus, Loki, Tempo, and the Kubernetes API,
and returns the results as a navigable graph.

Neighborhood Search
: Find everything reachable within N steps.
  _"Show me everything related to this pod"_ — returns owning deployments, logs, metrics, network flows, and more.

Goal Search
: Find paths to a specific type of data.
  _"Find all logs related to this alert"_ — korrel8r connects the alert to deployments, deployments to pods, pods to logs.

Korrel8r works with many query languages and storage systems, bridges
[OpenTelemetry](https://opentelemetry.io) and non-OTEL data,
and is extensible via YAML rules and pluggable domains.
It can be used as an in-cluster service and a local command.

- Read the [documentation](docs/introduction) to install and use korrel8r.
- There are some [recorded demos](https://github.com/korrel8r/demos) you can look at.
- See the [GitHub project](https://github.com/korrel8r/korrel8r) to contribute or learn more.
