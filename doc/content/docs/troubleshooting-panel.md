---
title: OpenShift Troubleshooting Panel
description: Correlation graphs in the OpenShift console
weight: 6
---

The OpenShift Console includes a [troubleshooting panel](https://github.com/openshift/troubleshooting-panel-console-plugin) plugin
that uses Korrel8r to display interactive correlation graphs.
When you view a resource in the console, the panel shows related observability signals
-- logs, metrics, alerts, traces, events, and more --
as a clickable graph that lets you navigate between them.

This is one important application of Korrel8r, but the [REST API](../reference/rest/) and
[MCP tools](../ai-agents/#mcp-tools-reference) can power other integrations too.
See [Other Uses](../other-uses/) for more ideas.

## How It Works

1. You view a resource in the OpenShift console (e.g. a Pod or Deployment).
2. The troubleshooting panel calls Korrel8r's REST API with a neighborhood or goal search.
3. Korrel8r applies its correlation rules, queries the relevant stores, and returns a graph.
4. The panel renders the graph as clickable nodes -- click any node to navigate to that data in the console.

The panel is read-only: it shows what Korrel8r found, but does not modify any cluster state.

## Installation

The troubleshooting panel is automatically installed and configured when you install the
[Cluster Observability Operator](../getting-started/#cluster-observability-operator) (COO)
on an OpenShift cluster. No additional configuration is needed.

For details on the COO and Korrel8r installation, see [Getting Started](../getting-started/).

## Using the Panel

For detailed usage instructions — opening the panel, navigating the correlation graph,
using focus, search settings, goal-directed searches, and status markers —
see the [Troubleshooting Panel User Guide](https://github.com/openshift/troubleshooting-panel-console-plugin/blob/main/doc/user-guide.md).

## AI Agent Integration

The troubleshooting panel can work together with an AI agent via Korrel8r's
[agent-console navigation](../ai-agents/#agent-console-navigation).
In this mode, an AI agent can see what you are viewing in the console and update the display
with its findings — enabling a conversational troubleshooting workflow.

See [AI Agents](../ai-agents/) for setting up the korrel8r side, and the
[Agent Navigation Guide](https://github.com/openshift/troubleshooting-panel-console-plugin/blob/main/doc/agent-navigation.md)
for enabling and using the feature in the console.

## Further Reading

- [Cluster Observability Operator documentation](https://docs.openshift.com/container-platform/latest/observability/cluster_observability_operator/cluster-observability-operator-overview.html)
- [Troubleshooting panel plugin](https://github.com/openshift/troubleshooting-panel-console-plugin)
