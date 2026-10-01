Prometheus metrics.

### Classes

```
metric:metric
```

### Object

A \[Metric\] is a time series identified by a label set. Korrel8r only uses labels for correlation, it does not use sample values. If a korrel8r search has time constraints, then metrics with no values that meet the constraint are ignored.

### Query

A selector is a [PromQL](<https://prometheus.io/docs/prometheus/latest/querying/basics/>) query string with optional trailing "namespace" parameters.

```
PROMQL[?namespace=NS[&namespace=NS...]]
```

Korrel8r parses the PromQL expression to identify the time series it refers to. It uses only metric labels for correlation, not time\-series data values.

The store's "metric" URL is the cluster\-wide metrics endpoint. The optional "namespaced" URL is a separate endpoint for namespace\-scoped access; it allows users to query their own namespaces without cluster\-wide metrics permissions. Korrel8r sends queries with a namespace suffix to this endpoint when it is configured, or to the "metric" URL otherwise.

For namespace\-scoped queries, use both a PromQL selector for the metric's namespace label and a matching "?namespace=NS" suffix. The selector filters time series; the suffix specifies the scope and selects the namespaced URL when available. The PromQL label is not always "namespace" \(for example "exported\_namespace"\) but the suffix is always "?namespace=NS".

Examples:

```
metric:metric:kube_pod_info{namespace="observability"}?namespace=observability
metric:metric:{exported_namespace="myapp",exported_name="web"}?namespace=myapp
```

### Store

Prometheus is the store. Store configuration:

```
domain: metric
metric: URL_OF_PROMETHEUS
namespaced: URL_OF_NAMESPACE_SCOPED_PROMETHEUS
```

