// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package alert

const Description = `

Alerts that metric values are out of bounds.

## Classes

    alert:alert

## Object

See the [Go Object type](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/domains/alert#Object).
Use capitalized Go field name in templates, not lowercase JSON names.

## Query

Selector is one of the following
- JSON object with alert label field names and matching label values.
- Array of objects as above, gets alerts that match any object in the array

An empty selector matches all alerts:

    alert:alert:{}

Examples:
    alert:alert:{"container":"kube-rbac-proxy-main","namespace":"openshift-logging"}
    alert:alert:[{"alertname":"alert1"},{"alertname":"alert2"}]

## Store

A client of Prometheus and/or AlertManager. Store configuration fields:

    domain: alert
    metrics: PROMETHEUS_URL
    alertmanager: ALERTMANAGER_URL

At least one of the fields "metrics" or "alertmanager" must be present.
`
