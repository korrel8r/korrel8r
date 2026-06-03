---
title: Client-Server Access
description: REST API and MCP access to korrel8r servers
weight: 5
---

To access the in-cluster service from outside the cluster, you need a `route` or `ingress`.
You can create one like this:

``` bash
oc create route reencrypt --service=korrel8r -n openshift-cluster-observability-operator
```

The _external_ URL for this route is:
```bash
oc apply -k github.com/korrel8r/korrel8r/config/route?version=main
export KORREL8R_URL=$(oc get route/korrel8r -n openshift-cluster-observability-operator -o template='https://{{.spec.host}}')
```

You can access the server in 2 ways:
- [`korrel8rcli`](https://korrel8r.github.io/client/)) — purpose-built command line client for Korrel8r
- Direct HTTP requests — use `curl` or similar tools against the [REST API](../reference/rest/)

### Command Line Client

[`korrel8rcli`](https://korrel8r.github.io/client/)  is a command line client to call on a remote `korrel8r` server.

**Installation:**
```bash
go install github.com/korrel8r/client/cmd/korrel8rcli@latest
```

`korrel8rcli` automatically uses your cluster login credentials.

Replace `$KORREL8R_URL` with the URL of your korrel8r service in these examples.

```bash
# Check what data sources are available
korrel8rcli -u $KORREL8R_URL domains

# Find everything related to a deployment
korrel8rcli -u $KORREL8R_URL neighbors --query 'k8s:Deployment:{namespace: korrel8r}'

# Find all logs related to a deployment
korrel8rcli -u $KORREL8R_URL goals --start 'k8s:Deployment:{namespace: korrel8r}' --goal 'log:application'
```

See the [documentation](https://korrel8r.github.io/client/) or run `korrel8rcli --help` for more details.

### Direct REST API Access

You can use `curl` or any HTTP client to interact with the Korrel8r REST API directly.
You need to pass a *bearer token* to the service, `$(oc whoami -t)` returns your token.

```bash
# Get available domains
curl --oauth2-bearer $(oc whoami -t) $KORREL8R_URL/api/v1alpha1/domains

# Perform a neighborhood search
curl --oauth2-bearer $(oc whoami -t) \
     "$KORREL8R_URL/api/v1alpha1/graphs/neighbors?depth=2&query=k8s:Pod:{namespace:default}"
```

See the complete [REST API Reference](../reference/rest/) for all available endpoints.

## Troubleshooting

> [!TIP]
> You can increase the verbosity of korrel8r logging at run time using the
> [config API](../reference/rest/#putconfig).

Using [`korrel8rcli`](https://korrel8r.github.io/client/):
```bash
korrel8rcli config --set-verbose=9
```

Using curl:
```bash
curl --oauth2-bearer $(oc whoami -t) -X PUT http://localhost:8080/api/v1alpha1/config?verbose=9
```
