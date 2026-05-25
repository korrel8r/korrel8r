---
title: Getting Started
weight: 2
---

# Getting Started

Korrel8r can run in different environments depending on your needs:

- **In-cluster**: Deployed as a Kubernetes service, recommended for production.
- **Out-of-cluster**:
  - **Command line**: direct execution of queries for automated scripts or tests.
  - **Service**: useful for testing, connect to cluster resources via routes.

See [Configuration](../reference/configuration/) for details on configuring stores and rules.

> **Authentication**: Korrel8r uses Bearer Tokens to authenticate with cluster stores.
>
> - As a service: Impersonates clients by forwarding their Bearer tokens
> - Command line: Uses your current `kubectl` login credentials

## Installation Options

### Cluster Observability Operator (recommended)

The [Red Hat Cluster Observability Operator](https://docs.openshift.com/container-platform/latest/observability/cluster_observability_operator/cluster-observability-operator-overview.html)
automatically installs and configures Korrel8r on OpenShift clusters,
configured to connect to the available stores and the console.
This is the easiest way to get up and running.

### Direct Deployment

This deploys korrel8r in the namespace `korrel8r` with a configuration suitable for OpenShift clusters.
Modify `configmap/korrel8r` to change the configuration.

Deploy the latest version:
```bash
oc apply -k github.com/korrel8r/korrel8r/config/?version=main
```

Deploy a specific version (replace X.Y.Z):
```bash
oc apply -k github.com/korrel8r/korrel8r/config/?version=vX.Y.Z
```

### Accessing the Service

To access Korrel8r from outside the cluster, create a route:

```bash
oc apply -k github.com/korrel8r/korrel8r/config/route?version=main
export KORREL8R_URL=$(oc get route/korrel8r -n korrel8r -o template='https://{{.spec.host}}')
curl --oauth2-bearer $(oc whoami -t) $KORREL8R_URL/api/v1alpha1/domains
```

## Running Outside the Cluster

You can run Korrel8r locally for development and testing.
This connects to cluster stores via routes or ingress.

**Step 1: Install the Korrel8r command**
```bash
go install github.com/korrel8r/korrel8r/cmd/korrel8r@latest
korrel8r --help  # View available commands
```

**Step 2: Download configuration for external access**
```bash
curl -o korrel8r.yaml https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-route.yaml
```

**Step 3: Run the service locally**
```bash
korrel8r -v2 --config korrel8r.yaml web --http=localhost:8080
```

Your local Korrel8r service is now available at `http://localhost:8080`.

## Using Korrel8r

Once your Korrel8r service is running, it provides a [REST API](../reference/rest/).
You can interact with it in several ways:

- **Command line client** ([`korrel8rcli`](/client/)) — Purpose-built for Korrel8r
- **Direct HTTP requests** — Direct HTTP calls using `curl` or similar tools
- **OpenShift Console** — Integrated troubleshooting panel
- **Web browser** — Built-in visualization (experimental)

> Clients require a _Bearer Token_ to authenticate.
> If you are logged in to a cluster, you can find your bearer token like this:
> ```bash
> oc whoami -t
> ```

### Command Line Client (`korrel8rcli`)

The [`korrel8rcli`](/client/) client provides a simple way to explore correlations from the command line.
See the [client documentation](/client/) for full details.

**Installation:**
```bash
go install github.com/korrel8r/client/cmd/korrel8rcli@latest
korrel8rcli --help  # View available commands
```

`korrel8rcli` automatically uses your cluster login credentials.
Replace `$KORREL8R_URL` with your URL.

#### Quick Examples

```bash
# Check what data sources are available
korrel8rcli -u $KORREL8R_URL domains

# Find everything related to a deployment
korrel8rcli -u $KORREL8R_URL neighbors --query 'k8s:Deployment:{namespace: korrel8r}'

# Find all logs related to a deployment
korrel8rcli -u $KORREL8R_URL goals --start 'k8s:Deployment:{namespace: korrel8r}' --goal 'log:application'
```

### Direct REST API Access

You can use `curl` or any HTTP client to directly interact with the Korrel8r REST API:

```bash
# Get available domains
curl --oauth2-bearer $(oc whoami -t) $KORREL8R_URL/api/v1alpha1/domains

# Perform a neighborhood search
curl --oauth2-bearer $(oc whoami -t) \
     "$KORREL8R_URL/api/v1alpha1/graphs/neighbors?depth=2&query=k8s:Pod:{namespace:default}"
```

See the complete [REST API Reference](../reference/rest/) for all available endpoints.

### OpenShift Console Integration

The OpenShift Console includes a troubleshooting panel that uses Korrel8r to display clickable correlation graphs.
When viewing resources in the console, you can navigate between related logs, metrics, alerts, and other observability data.

### Data Access Client

Korrel8r can be used as a client to directly access data from any of the stores it can connect to.
There are existing clients for most korrel8r domains, such as promcli (alert, metric), logcli (log), kubectl (k8s) etc.

Korrel8r does not have all the features of every native client, but it can be useful as a simplified _uniform_ client:

- A single tool for all the supported stores.
- Can run as a command line tool, or deploy as a service with REST and MCP APIs.
  - As a service it provides token-forwarding to integrate securely with Kubernetes and MCP tools.
- Uses the same well-known native query languages as native tools (e.g. PromQL or LogQL).
  - Korrel8r queries just add a prefix to native query strings.

The output format of korrel8r differs slightly from some native tool output.

## Troubleshooting

> **Tip**: You can increase the verbosity of korrel8r logging at run time using the
> [config API](../reference/rest/#putconfig).

Using [`korrel8rcli`](/client/):
```bash
korrel8rcli config --set-verbose=9
```

Using curl:
```bash
curl --oauth2-bearer $(oc whoami -t) -X PUT http://localhost:8080/api/v1alpha1/config?verbose=9
```
