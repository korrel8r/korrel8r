---
title: AI Agents
description: Using korrel8r with AI agents via MCP
weight: 7
---

Korrel8r gives AI agents the ability to navigate observability signals
and Kubernetes resources across an entire cluster,
without needing to know the query language, API, or data model of each store.

It integrates with agents via the [Model Context Protocol](https://modelcontextprotocol.io/) (MCP),
providing tools to explore correlations, retrieve data, and interact with the OpenShift console.

There are two main ways agents use korrel8r:

1. [Correlation and data access](#correlation-and-data-access) — agent can find related signals and retrieve observability data across domains.
2. [Agent-console navigation](#agent-console-navigation) — two-way link between an AI agent and the OpenShift console.

## Correlation and data access

### Reducing token costs

Korrel8r returns [correlation graphs](../introduction/#correlation-graphs) containing queries, counts,
and [statuses](../statuses/) — not full data objects.
An agent can examine what data is available and how interesting it is _before_ making expensive calls
to retrieve the actual data, significantly reducing token costs.
See [Graph-first workflow](../introduction/#graph-first-workflow) for details.

### Discovering available data

The `list_domains` tool returns the available [domains](../introduction/#domains-organize-data)
(logs, metrics, alerts, traces, Kubernetes resources, etc.).
The `list_domain_classes` tool lists the [classes](../introduction/#domains-organize-data) within a domain.
The `help` tool returns documentation for a domain, including query syntax and examples.

See the [Domain Reference](../reference/domains/) for all domains and their query syntax.

### Searching for correlations

Korrel8r offers two search strategies, exposed as MCP tools:

`create_goals_graph`
: Find paths from a starting point to a specific kind of data.
  Use this for targeted questions like "find logs related to this pod"
  or "what alerts fired for this deployment?"
  See [Goal Search](../introduction/#goal-search) for how this works.

`create_neighbors_graph`
: Explore everything related to a starting point, up to a given depth.
  Use this for open-ended investigation like "what is related to this pod?"
  or "show me everything connected to these alerts."
  See [Neighborhood Search](../introduction/#neighborhood-search) for how this works.

Both tools return a correlation graph — nodes representing classes of data (with queries and result counts)
and edges representing the correlation rules that connect them.

### Retrieving data

The `get_objects` tool executes a [query](../introduction/#domains-organize-data) and returns matching objects as JSON.
The agent can use queries from the correlation graph, or construct its own.
An optional `constraint` parameter limits results by time range and/or count.

### Example: investigating a crashing pod

A typical agent session might look like this:

1. The agent calls `help` with domain `k8s` to learn the query syntax.
2. The agent calls `create_goals_graph` with start query `k8s:Pod:{"namespace":"myapp","name":"web-0"}` and goals `["log:application"]`.
3. Korrel8r returns a graph showing the path: Pod → logs, with queries and result counts.
4. The agent calls `get_objects` with the log query from the graph (adding a time constraint) to retrieve the actual log entries.
5. The agent analyzes the logs and reports the root cause.

## Agent-console navigation

*When you need to: let an agent see what a user is viewing in the OpenShift console, or update the console to show relevant data.*

For the console side see the [Console Agent Navigation Guide](https://github.com/openshift/troubleshooting-panel-console-plugin/blob/main/doc/agent-navigation.md).

Korrel8r connects an AI agent (connected via MCP) and the OpenShift console (connected via REST),
enabling conversational troubleshooting: the user looks at something in the console,
asks the agent a question, and the agent can understand the context and display its findings back in the console.

The connection works through a shared [session](../reference/rest/#putconsole).
The console and the agent authenticate with the same bearer token to share a session.

### Console → Agent: reading console state

The console sends its current state to korrel8r via `PUT /console` (REST API).
This includes:
- **view**: a korrel8r query describing what the main console view is displaying.
- **search**: the correlation search parameters shown in the troubleshooting panel.

The agent reads this state by calling the `get_console` MCP tool.

### Agent → Console: updating the display

The agent can update the console by calling the `show_in_console` MCP tool with:
- **view**: a query to change what the main console view displays.
- **search**: correlation search parameters to display in the troubleshooting panel.

The console receives these updates in real time via `GET /console/events` (an SSE stream from the REST API).

### Example: conversational troubleshooting

1. A user is viewing a deployment in the OpenShift console and sees it is unhealthy.
2. The user asks the agent: "Why is this deployment failing?"
3. The agent calls `get_console` and sees the user is looking at `k8s:Deployment.apps:{"namespace":"myapp","name":"web"}`.
4. The agent calls `create_neighbors_graph` to explore related signals.
5. The agent finds relevant error logs and a firing alert, retrieves and analyzes them.
6. The agent calls `show_in_console` with a query to display the relevant logs in the console.
7. The user sees the logs appear in the console and can investigate further.

## Setup and configuration

### Prerequisites

- An OpenShift cluster with observability stores deployed (Prometheus, Loki, etc.)
- Korrel8r installed — see [Getting Started](../getting-started/) for installation options
- An MCP-capable AI agent

### Installing korrel8r

Follow the [Getting Started](../getting-started/) guide.
For OpenShift, the recommended approach is the
[Cluster Observability Operator](../getting-started/#cluster-observability-operator),
which deploys and configures korrel8r automatically.

### Connecting an agent via MCP stdio

For agents that launch MCP servers as subprocesses (Claude Code, Claude Desktop, etc.),
use the [`korrel8r mcp`](../reference/cmd/korrel8r_mcp/) command.

Configure your agent's MCP settings to launch korrel8r:

```json
{
  "mcpServers": {
    "korrel8r": {
      "command": "korrel8r",
      "args": ["--config", "/path/to/korrel8r.yaml", "mcp"]
    }
  }
}
```

Korrel8r uses the current `kubectl`/`oc` login credentials to access the cluster.

### Connecting an agent via MCP HTTP

For agents that connect to a remote MCP server,
korrel8r serves the MCP Streamable HTTP protocol at `/mcp` when running in web mode.

```bash
korrel8r web --http :8080
```

The agent connects to `http://<host>:8080/mcp` and authenticates with a bearer token.
This mode also serves the REST API at `/api/v1alpha1`, required for agent-console navigation.
See [`korrel8r web`](../reference/cmd/korrel8r_web/) for all options.

### Enabling agent-console navigation

Korrel8r must run as a web service (not stdio) with both MCP (`/mcp`) and REST (`/api/v1alpha1`) endpoints enabled (the default).
The console and agent must authenticate as the same user to share a session.

> [!NOTE]
> The console and agent bearer tokens must belong to the same user, but don't need to be identical.
> Korrel8r uses `tokenreviews` to determine the user associated with bearer tokens.

For the full console setup see the console
[Agent Navigation Guide](https://github.com/openshift/troubleshooting-panel-console-plugin/blob/main/doc/agent-navigation.md).

### Example: Claude Code with local korrel8r

Claude Code as the AI agent with korrel8r running locally, connected to a remote OpenShift cluster.

**Step 1: Log in to the cluster**
```bash
oc login <cluster-url>
```

**Step 2: Download the configuration for external access**
```bash
curl -o korrel8r.yaml https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-route.yaml
```

**Step 3: Configure Claude Code**

Add korrel8r as an MCP server in `.claude/settings.json` or project settings:

```json
{
  "mcpServers": {
    "korrel8r": {
      "command": "korrel8r",
      "args": ["--config", "korrel8r.yaml", "mcp"]
    }
  }
}
```

**Step 4: Use it**

Claude Code will automatically discover korrel8r's MCP tools.
You can ask questions like:
- "What domains does korrel8r know about?"
- "Find all logs related to the deployment `web` in namespace `myapp`"
- "What is related to this pod? Show me everything within 2 steps."

### Example: agent-console navigation

See the [Agent Navigation Guide](https://github.com/openshift/troubleshooting-panel-console-plugin/blob/main/doc/agent-navigation.md)
for step-by-step instructions on connecting an agent to the console.

For development and testing, the `--unsafe-shared-session` flag
lets the console and agent share a session without requiring matching bearer tokens:

```bash
korrel8r web --http :8080 --unsafe-shared-session
```

> [!WARNING]
> `--unsafe-shared-session` means all users share the same session.
> Do not use this in production.

## MCP tools reference

See also the [REST API Reference](../reference/rest/) and [Domain Reference](../reference/domains/).

| Tool | Description |
|------|-------------|
| `list_domains` | List available [domains](../introduction/#domains-organize-data) with descriptions |
| `list_domain_classes` | List [classes](../introduction/#domains-organize-data) within a domain |
| `help` | Get documentation and query syntax for a domain (or all domains) |
| `create_goals_graph` | [Goal search](../introduction/): find paths from start objects to specific goal classes |
| `create_neighbors_graph` | [Neighborhood search](../introduction/): explore all data reachable within N steps |
| `get_objects` | Execute a [query](../introduction/#domains-organize-data) and return matching objects |
| `get_console` | Read the current console state (for [agent-console navigation](#agent-console-navigation)) |
| `show_in_console` | Update the console display (for [agent-console navigation](#agent-console-navigation)) |
