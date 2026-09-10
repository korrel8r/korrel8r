---
title: Configuring Stores
description: Connecting korrel8r to the observability stores in your cluster
weight: 8
---

A _store_ is korrel8r's client connection to a back-end that holds observability data.
Korrel8r needs at least one store per [domain](../reference/domains/) it should search.
If a domain has no store, korrel8r still knows its rules, but cannot retrieve any of its data.

The built-in configurations assume every store is installed in its default OpenShift location.
Pointing korrel8r at a store somewhere else is the most common configuration change, and is what this page covers.
For the full config file syntax, see the [Configuration reference](../reference/configuration/).

## Stores in the built-in configuration

Korrel8r ships with two configurations that differ only in _how they reach_ the same set of stores:

[openshift-svc.yaml](https://github.com/korrel8r/korrel8r/blob/main/etc/korrel8r/openshift-svc.yaml)
: For korrel8r running **in** the cluster. Uses in-cluster service URLs.
  This is the default for the deployed service.

[openshift-route.yaml](https://github.com/korrel8r/korrel8r/blob/main/etc/korrel8r/openshift-route.yaml)
: For korrel8r running **outside** the cluster. Looks up each store's host from its OpenShift Route.

Both configure these stores:

| Domain | Back-end | Store field | Default location |
|--------|----------|-------------|------------------|
| `k8s` | Kubernetes API server | *(none)* | Your current kubectl/oc login |
| `log` | LokiStack | `lokiStack` | `logging-loki-gateway-http` in `openshift-logging` |
| `metric` | Thanos / Prometheus | `metric` | `thanos-querier` in `openshift-monitoring` |
| `alert` | Thanos + Alertmanager + Loki ruler | `metrics`, `alertmanager`, `lokiRuler` | `openshift-monitoring`, plus `openshift-logging` for the ruler |
| `netflow` | LokiStack (NetObserv) | `lokiStack` | `loki-gateway-http` in `netobserv` |
| `trace` | TempoStack | `tempoStack` | `tempo-platform-gateway` in `openshift-tracing` |
| `incident` | Thanos / Prometheus | `metrics` | `thanos-querier` in `openshift-monitoring` |

## Check which stores are working

Before changing anything, find out what korrel8r currently sees:

```bash
korrel8r domains                       # Domains and stores in the local configuration file
korrel8rcli -u $KORREL8R_URL domains   # Domains and stores used by a remote server
```

A store that failed to connect is reported with an `error` field, which is usually enough to tell
whether the URL is wrong, the store is not installed, or the connection was refused.

## Replace default stores

To point a domain at a different back-end, write your own configuration file with a `stores` entry
for that domain, and `include` the built-in rules so you keep all the correlation logic:

```yaml
# my-korrel8r.yaml
stores:
  - domain: k8s
  - domain: metric
    metric: https://my-prometheus.example:9090

include:
  - /etc/korrel8r/rules/all.yaml
```

Run with it:

```bash
korrel8r web --config my-korrel8r.yaml
```

Only the domains you list get stores. The example above configures `k8s` and `metric` only, so
searches in every other domain return nothing. To change one store but keep the rest, include a
built-in configuration instead of `rules/all.yaml` — see
[Add a store to the defaults](#add-a-store-to-the-defaults).

## Add a store to the defaults

A domain can have more than one store. Korrel8r queries **all** stores for a domain and combines
the results, so adding an entry supplements the default rather than replacing it.

Include a built-in configuration, then list the extra store:

```yaml
# my-korrel8r.yaml
include:
  - /etc/korrel8r/openshift-svc.yaml   # All the default stores and rules

stores:
  - domain: metric
    metric: https://my-other-prometheus.example:9090
```

Searches in the `metric` domain now query both the cluster Thanos and your own Prometheus.

## Connect over TLS

Stores served with a private CA need a `certificateAuthority` field giving the path to a CA
certificate file. In-cluster, this is the service CA bundle mounted into every pod:

```yaml
stores:
  - domain: metric
    metric: https://thanos-querier.openshift-monitoring.svc:9091
    certificateAuthority: /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
```

## Look up a store URL dynamically

Store fields are [Go templates](../reference/configuration/#about-templates), so a URL can be
computed from the cluster instead of hard-coded. This is how `openshift-route.yaml` finds stores
when korrel8r runs outside the cluster:

```yaml
stores:
  - domain: log
    lokiStack: 'https://{{k8sRouteHost "openshift-logging" "logging-loki"}}'
```

`k8sRouteHost` returns the host of the named Route. See the
[template functions reference](../reference/template-functions/) for the functions available.

## Apply a custom configuration in the cluster

For the deployed service, put your configuration in a ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: korrel8r-custom-config
data:
  korrel8r.yaml: |
    include:
      - /etc/korrel8r/openshift-svc.yaml
    stores:
      - domain: metric
        metric: https://my-prometheus.example:9090
```

Mount it in the deployment and point `--config` at it.
Mount at a sub-directory such as `/etc/korrel8r/custom/`, _not_ at `/etc/korrel8r` itself: mounting
over that directory would hide the built-in configurations and rules that your `include` needs.

```yaml
volumes:
  - name: custom-config
    configMap:
      name: korrel8r-custom-config
containers:
  - name: korrel8r
    command: ["korrel8r", "web", "--config=/etc/korrel8r/custom/korrel8r.yaml"]
    volumeMounts:
      - name: custom-config
        mountPath: /etc/korrel8r/custom
        readOnly: true
```

## Next steps

- [Domain reference](../reference/domains/) — every store field accepted by each domain.
- [Configuration reference](../reference/configuration/) — the complete config file format.
