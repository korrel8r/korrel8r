---
title: netflow
description: network flow data.
---
network flow data.

### Classes

```
netflow:network
```

### Object

JSON object in [NetFlow](<https://docs.openshift.com/container-platform/latest/observability/network_observability/json-flows-format-reference.html>) format.

### Query

Selector is a [LogQL](<https://grafana.com/docs/loki/latest/query/>) query string. Examples:

```
netflow:network:{SrcK8S_Type="Pod", SrcK8S_Namespace="myNamespace"}
netflow:network:{DstK8S_Namespace="openshift-apiserver", DstK8S_OwnerName="apiserver"}
```

### Store

To connect to a netflow lokiStack store use this configuration:

```
domain: netflow
lokistack: URL_OF_LOKISTACK_PROXY
```

To connect to plain loki store use:

```
domain: netflow
loki: URL_OF_LOKI
```

