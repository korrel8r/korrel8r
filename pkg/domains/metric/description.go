// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

const Description = `

## Classes

    metric:metric

## Object

A [Metric](https://pkg.go.dev/github.com/prometheus/common@v0.45.0/model#Metric) is a time series identified by a label set. Korrel8r only uses labels for correlation, it does not use sample values. If a korrel8r search has time constraints, then metrics with no values that meet the constraint are ignored.

## Query

Selector is a [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query string.

Korrel8r uses metric labels for correlation, it does not use time-series data values.
The PromQL expression is parsed to extract the label matchers for the series it refers to.

Examples:

    metric:metric:kube_pod_info{namespace="default"}
    metric:metric:{namespace="tracing-app-k6",pod="k6-tracing-564cf6dc8b-hpxd2"}

## Store

Prometheus is the store, store configuration:

    domain: metric
    metric: URL_OF_PROMETHEUS
`
