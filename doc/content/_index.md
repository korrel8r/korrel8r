---
title: Korrel8r
weight: 1
description: Rule-based correlation engine for Kubernetes
hideChildren: true
---

Korrel8r is a rule-based correlation engine for Kubernetes clusters.

It connects cluster resources and observability signals — logs, metrics, alerts, traces, network flows — into a graph of related data
by searching multiple back-ends with different APIs and query languages automatically.

Given any starting point — an alert, a pod, a deployment — it finds related data across multiple signal stores,
such as Prometheus, Loki, Tempo, and the Kubernetes API.

Neighborhood Search
: Find everything reachable within N steps.
  _"Show me everything related to this pod"_ — returns owning deployments, logs, metrics, network flows, and more.

Goal Search
: Find paths to a specific type of data.
  _"Find all logs related to this alert"_ — korrel8r connects the alert to deployments, deployments to pods, pods to logs.

### Use cases

Interactive troubleshooting
: The [OpenShift troubleshooting panel](https://docs.redhat.com/en/documentation/red_hat_openshift_cluster_observability_operator/1-latest/html/ui_plugins_for_red_hat_openshift_cluster_observability_operator/troubleshooting-ui-plugin) displays interactive correlation graphs using Korrel8r as a back-end. Click to jump to relevant resources directly, without manually discovering the intermediate relationships.

AI-agent troubleshooting
: Korrel8r provides an [MCP](https://korrel8r.github.io/korrel8r/docs/reference/mcp/) interface for AI agents. Its rule-based engine searches and correlates large data sets faster than an AI agent can. It builds a small, focused graph of relevant signals, so the AI can concentrate on understanding the problem rather than spending tokens discovering connections.

Custom automation
: Korrel8r provides REST and MCP interfaces, can run as a cluster service or a command, and can be used as a Go library.
Embed it in custom automation solutions that need correlation data.

### Extending

Rules
: Take data from a "start" class to generate a "goal" query for related data.
Rules are written as YAML files using Go template syntax.

Domains
: A "domain" defines a signal type, query language, and data store.
New domains can be added to the engine. Coding is required.

