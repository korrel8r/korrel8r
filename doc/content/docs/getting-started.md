---
title: Getting Started
description: Installation and configuration
weight: 3
---

Korrel8r can be deployed as a service in a cluster, or used from the command line to examine remote cluster data. 

## Deployment options

### Cluster Observability Operator

Installing the [Red Hat Cluster Observability Operator](https://docs.openshift.com/container-platform/latest/observability/cluster_observability_operator/cluster-observability-operator-overview.html) is the easiest way to get started.

It deploys Korrel8r in your cluster, configured to connect to alerts, metrics, logging, traces and network events.
The [OpenShift Console Troubleshooting Panel](../troubleshooting-panel/) displays searches as interactive graphs.

### Deploy latest images

You can deploy the latest (unsupported) development images to namespace `korrel8r`.
```bash
oc apply -k github.com/korrel8r/korrel8r/config/?version=main
```

>[!NOTE]
> This uses a default configuration designed for OpenShift, for other clusters you will need to modify the configuration. See [Configuration](../configuration/).

## Command Line

To install:
```bash
go install github.com/korrel8r/korrel8r/cmd/korrel8r@latest
```

Download the sample configuration file:
```bash
curl -o korrel8r.yaml https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-route.yaml
```

> [!NOTE]
> This configuration works with OpenShift clusters with the default routes to observability stores.
> For other clusters you may need to modify the default configuration.

Set an environment variable to avoid repeating the configuration file:
``` bash
export KORREL8R_CONFIG=$PWD/openshift-route.yaml
```

You can use `korrel8r --config <file-location>` instead of the environment variable.

## Next Steps

Now that korrel8r is running:

- [Command Line](../command/) — using the command line tool.
- [Client Access](../client/) — client access to a remote Korrel8r server.
- [OpenShift Troubleshooting Panel](../troubleshooting-panel/) — use korrel8r in the OpenShift console
- [AI Agents](../ai-agents/) — connect an AI agent via MCP
