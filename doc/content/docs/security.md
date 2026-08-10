---
title: Security
description: Security notes
weight: 99
---

## Authentication

Korrel8r does not authorize requests itself.
It forwards the client's Kubernetes bearer token to backend stores (Prometheus, Loki, Alertmanager, the Kubernetes API, etc.), which enforce their own access control.

- **Cluster service** (REST or MCP over HTTP): the client provides a bearer token in the `Authorization` header. Korrel8r validates it via Kubernetes [TokenReview](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-review-v1/) to identify the user, and forwards it on backend requests.
- **Command line**: uses the logged-in user's token (equivalent to `oc whoami -t`).

Non-admin users are automatically routed through namespace-scoped tenancy proxies for Prometheus and Alertmanager.

## Session isolation

As a cluster service, korrel8r creates a separate session per authenticated user, keyed by the username.
Each session has its own engine instance, console state, and SSE event stream.
Sessions expire after a configurable idle timeout.

## TLS

REST and MCP HTTP connections are TLS-secured when running as a cluster service.
The OpenShift deployment uses automatically provisioned serving certificates.
TLS version, cipher suites, and curves are configurable via CLI flags.

Connections to backend stores use the cluster service-account CA.

## MCP transports

**Stdio** (`korrel8r mcp`)
: No network authentication — the process boundary provides isolation. Uses the logged-in user's token for store access.

**Streamable HTTP** (`korrel8r web --mcp`)
: Served at `/mcp` behind the same session middleware as the REST API. Each client gets its own session based on its bearer token.

## Browser and AI coordination

The OpenShift console plugin and an AI agent can coordinate through a shared korrel8r session:

- The **browser** reports its current view to korrel8r via `PUT /console` (REST).
- The **AI agent** reads that state via the `get_console` MCP tool.
- The **AI agent** can push navigation updates via the `show_in_console` MCP tool.
- The **browser** receives those updates over an SSE stream at `GET /console/events`.

Coordination is session-scoped — the browser and the AI agent must authenticate as the same user.

The "view" data contains parsed and validated queries for back-end stores — no primary data.
The only browser action is to change the currently viewed page to one of a known set of views;
page details are populated by the browser and not visible to Korrel8r.
No other action or URL injection is possible.
