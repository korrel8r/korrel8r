// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package incident

const Description = `

Incidents group alerts into higher-level groups.

For more about incidents see the [cluster-health-analyzer](https://github.com/openshift/cluster-health-analyzer).

## Classes

    incident:incident

## Object

An incident object contains id and mapping to the sources.
Alert is the only supported source type at present.

## Query

Query selectors are JSON objects.

An empty selector matches all incidents:

    incident:incident:{}

Getting an incident by ID:

    incident:incident:{"id":"id-of-the-incident"}

Using alert labels to get a corresponding incident:

    incident:incident:{"alertLabels":{"alertname":"AlertmanagerReceiversNotConfigured","namespace":"openshift-monitoring"}}

## Store

A client of Prometheus. Store configuration:

    domain: incident
    metrics: PROMETHEUS_URL
`
