---
title: AI Agents
description: Using korrel8r with AI agents via MCP
weight: 7
---

Korrel8r integrates with AI agents via the [Model Context Protocol](https://modelcontextprotocol.io/) (MCP),
providing tools for correlation search, data retrieval, and OpenShift console navigation.
See the [MCP reference](../reference/mcp/) for details of tools.

## Openshift Console navigation

> [!WARNING]
> **Dev Preview**: AI Agent navigation is a developer preview feature, subject to change without notice.

Korrel8r connects an AI agent (via MCP) and the OpenShift console (via REST) through a shared session,
enabling conversational troubleshooting: the user looks at something in the console, asks the agent a question,
and the agent can understand the context and display findings back in the console.

> [!IMPORTANT]
> The console and agent must authenticate as the _same user_ to share a session.
> The tokens don't have to be identical but they must belong to the same user.

### Prerequisites

- An OpenShift cluster with Lightspeed, Cluster Observability Operator, and desired observability stores (Prometheus, Loki, etc.)
- Korrel8r installed via the Cluster Observability Operator — see [Getting Started](../getting-started/)

### Enabling agent navigation in the OpenShift console

**1. Enable the feature gate** — edit the `troubleshooting-panel` UIPlugin:

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: troubleshooting-panel
spec:
  type: TroubleshootingPanel
  troubleshootingPanel:
    enableAgentNavigation: true
```

**2. Create a route** to expose the in-cluster korrel8r service:

```bash
oc create route reencrypt --service=korrel8r -n openshift-cluster-observability-operator
export KORREL8R_URL=$(oc get routes/korrel8r -n openshift-cluster-observability-operator -o template='https://{{.spec.host}}')
```

**3. Configure the agent** to connect via MCP Streamable HTTP, authenticating as the same user as the console session:

```bash
export TOKEN=$(oc whoami -t)
```

```json
{
  "mcpServers": {
    "korrel8r": {
      "type": "streamable-http",
      "url": "<KORREL8R_URL>/mcp",
      "headers": {
        "Authorization": "Bearer <TOKEN>"
      }
    }
  }
}
```

**4. Enable in the console** — click the AI icon in the troubleshooting panel toolbar and toggle the switch on.
The icon color indicates status: green = connected, red = connection error (details in the AI menu).

> [!IMPORTANT]
> The bearer token and console login must belong to the same user.

### Example: conversational troubleshooting

1. User views an unhealthy deployment in the console and asks: _"Why is this failing?"_
2. Agent calls `get_console` — sees `k8s:Deployment.apps:{"namespace":"myapp","name":"web"}`.
3. Agent calls `create_neighbors_graph` to explore related signals.
4. Agent finds error logs and a firing alert, retrieves and analyzes them.
5. Agent calls `show_in_console` to display the relevant logs in the console.

## Other integrations

See the [MCP reference](../reference/mcp/) for details of the available tools.

### Connecting via MCP HTTP

korrel8r serves Streamable HTTP at `/mcp` in web mode:

```bash
korrel8r web --http :8080
```

The agent connects to `http://<host>:8080/mcp` with a bearer token.
This mode also serves the REST API at `/api/v1alpha1`, required for console navigation.
See [`korrel8r web`](../reference/cmd/korrel8r_web/) for options.

### Connecting via MCP stdio

For agents that launch MCP servers as subprocesses (Claude Code, Claude Desktop, etc.),
use [`korrel8r mcp`](../reference/cmd/korrel8r_mcp/):

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

Korrel8r uses the current `kubectl`/`oc` login credentials.

### Example: Claude Code with local korrel8r

```bash
oc login <cluster-url>
curl -o korrel8r.yaml https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-route.yaml
```

Add to `.claude/settings.json`:
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

Claude Code will discover korrel8r's tools automatically. Try:
- _"What domains does korrel8r know about?"_
- _"Find all logs related to deployment `web` in namespace `myapp`"_
- _"What is related to this pod? Show me everything within 2 steps."_

