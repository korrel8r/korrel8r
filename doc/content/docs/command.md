---
title: Command Line
description: Using korrel8r from the command line
weight: 4
---

From the command line, Korrel8r can:
- Get data from any of the cluster stores it is connected to.
- Get JSON correlation graphs of related queries.
- Run as an MCP tool (using the 'stdio' protocol) allowing a local AI agent to query the cluster.

## Simple examples

**Informational queries**
```bash
# List available domains
korrel8r list

# List classes in the k8s domain
korrel8r list k8s

# Get documentation for a domain
korrel8r describe k8s

```

**Universal client - resources, logs, metrics, etc.**
``` bash
# Print all pods in the default namespace.
korrel8r objects 'k8s:Pod:{namespace: "default"}'

# Print infrastructure logs from the API server.
korrel8r objects 'log:infrastructure:{kubernetes_namespace_name="openshift-kube-apiserver"}'

# Print metrics for the API server.
korrel8r objects 'metric:metric:apiserver_request_total'
```

**Correlation graphs**
``` bash
# Get a JSON graph of data related to a deployment within 2 steps
korrel8r neighbors -q 'k8s:Deployment.apps:{namespace: myapp, name: web}' --depth 2

# Get a JSON graph of paths from a deployment to application logs.
korrel8r goals -q 'k8s:Deployment.apps:{namespace: myapp, name: web}' log:application
```

**MCP tool**
Configure your agent to call this command as an MCP tool:
```
korrel8r mcp
```

See the [command reference](../reference/cmd/) for all options.

