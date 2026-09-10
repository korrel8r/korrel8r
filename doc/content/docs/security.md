---
title: Security for REST and MCP
description: Authentication, TLS, and session isolation for the korrel8r server
weight: 99
---

Korrel8r has no user database, permission model, or access rules of its own.
Clients authenticate with their normal Kubernetes bearer token, which korrel8r forwards to every
back-end store. Each store applies its own RBAC, so a client can never see data through korrel8r
that it could not retrieve from the store directly.

## Authentication and authorization

Clients authenticate with a cluster bearer token — the same OAuth token as `oc whoami -t` — in the
standard header on every REST or MCP request:

```
Authorization: Bearer <TOKEN>
```

Korrel8r uses the token for two things:

1. **Identifying the user.** A Kubernetes
   [TokenReview](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-review-v1/)
   resolves the token to a username. Missing or invalid tokens are rejected.
2. **Reaching the stores.** Requests to Prometheus, Loki, Alertmanager, Tempo and the Kubernetes
   API carry the client's own token, so the client's cluster permissions apply unchanged.

Non-admin users are routed through the namespace-scoped tenancy ports for Prometheus and
Alertmanager — see [Admin vs Regular User](../reference/user-access/).

On the command line there is no login step: korrel8r uses your current `oc` or `kubectl`
credentials.

## TLS

TLS protects requests and results in transit. Serve over HTTPS with a certificate and key:

```bash
korrel8r web --https :8443 --cert tls.crt --key tls.key
```

`--cert` and `--key` are both required. Three options constrain the negotiated encryption:

| Option | Purpose |
|--------|---------|
| `--tls-min-version VERSION` | Minimum TLS version, e.g. `VersionTLS12`, `VersionTLS13`. |
| `--tls-cipher-suites LIST` | Allowed cipher suites, IANA or OpenSSL names. |
| `--tls-curves LIST` | Allowed curves, Go or OpenSSL names, e.g. `CurveP256`, `X25519`. |

`--http ADDRESS` serves plain HTTP, sending tokens and results in the clear — local development
only, and incompatible with the TLS options above.
The OpenShift deployment uses HTTPS with an automatically provisioned serving certificate.

## Trusting store back-ends

Korrel8r's connection to each store is separate from the client's connection to korrel8r, and needs
its own trust settings. By default korrel8r uses the ambient cluster CA; a store presenting a
certificate from a different authority needs a `certificateAuthority` field naming a PEM CA file
that korrel8r can read:

```yaml
stores:
  - domain: metric
    metric: https://my-prometheus.example:9090
    certificateAuthority: /etc/korrel8r/certs/my-ca.crt
```

Trust is per-store, so each back-end can use a different CA. In-cluster stores normally use the
service CA bundle mounted into every pod,
`/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt`.
See [Configuring Stores](../configuring-stores/).

## MCP

**Streamable HTTP** (`korrel8r web`)
: Served at `/mcp`, enabled by default and disabled with `--mcp=false`. Token forwarding, TLS and
  sessions work exactly as for REST. MCP has no login step, so the client must send the bearer
  token itself — for most agents, as a header in the MCP server entry:

  ```json
  {
    "mcpServers": {
      "korrel8r": {
        "url": "<KORREL8R_URL>/mcp",
        "headers": { "Authorization": "Bearer <TOKEN>" }
      }
    }
  }
  ```

  That file holds a live cluster token: treat it as a secret, and refresh it when the token expires.

**Stdio** (`korrel8r mcp`)
: No network and no HTTP authentication — the process boundary is the security boundary. Store
  access uses your own credentials, so the agent has exactly your permissions.

See [AI Agents](../ai-agents/) for full client setup.

## Session isolation

As a server, korrel8r keeps one session per authenticated user, **keyed by the username** from the
bearer token. Each session has its own engine, console state and event stream, so concurrent users
cannot read or disturb each other's work.

Keying on username rather than token means one user with two valid tokens shares a single session,
while two users always get separate sessions, even from the same machine.

Sessions are discarded after an idle timeout set by
[`tuning.sessionTimeout`](../reference/configuration/#tuning).

## Browser and AI agent coordination

The OpenShift console plugin and an AI agent can coordinate through a shared korrel8r session:

- The **browser** reports its current view to korrel8r via `PUT /console` (REST).
- The **AI agent** reads that state via the `get_console` MCP tool.
- The **AI agent** can push navigation updates via the `show_in_console` MCP tool.
- The **browser** receives those updates over an SSE stream at `GET /console/events`.

Coordination is session-scoped: browser and agent must authenticate as the same user, though not
necessarily with the same token.

The shared view data contains parsed, validated store queries, never primary data. The agent's only
action is to switch the browser to one of a known set of views; page details are filled in by the
browser and not visible to korrel8r. No other action or URL injection is possible.
