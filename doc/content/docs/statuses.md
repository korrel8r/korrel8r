---
title: Statuses
description: How status rules attach statuses to correlation graph nodes
weight: 10
---

A _status_ is an UpperCamelCase string (no spaces or punctuation) that summarizes
the "interestingness" of data in a [correlation graph](../introduction/#correlation-graphs).
Each node can carry statuses like `Error`, `Warning`, or `Finalizer`,
with counts showing how many objects matched.
This lets you see which nodes have problems without retrieving the full data.

## How statuses work

_Status rules_ in YAML [configuration](../configuration/) files define how statuses are generated.
A status rule applies a [Go template](../configuration/#about-templates) to each object retrieved during a search.
The template outputs zero or more statuses (one per line); blank lines are ignored.
Korrel8r counts how many objects produce each status and attaches the counts to the graph node.

```yaml
statusRules:
  - name: RuleName
    start:
      domain: domain-name
      classes:               # optional — omit to apply to all classes in the domain
        - ClassName
    status: |-
      template-that-outputs-statuses
```

The `start` field works the same as in [correlation rules](../writing-rules/#rule-structure).

## Built-in status rules

### Log severity

Classifies log entries as `Error` or `Warning` based on the `level` or `severity_text` field.

```yaml
statusRules:
  - name: LogSeverity
    start:
      domain: log
    status: |-
      {{- $s := or (index . "level") (index . "severity_text") ""}}
      {{- if or (eq $s "error") (eq $s "err") (eq $s "ERROR") ...}}Error
      {{- else if or (eq $s "warning") (eq $s "warn") ...}}Warning
      {{- end}}
```

### Alert severity

Extracts the severity from alerts (e.g. `Critical`, `Warning`).

```yaml
statusRules:
  - name: AlertSeverity
    start:
      domain: alert
    status: |-
      {{with .Labels.severity}}{{if ne . "none"}}{{.}}{{end}}{{end}}
```

### Kubernetes event type

Marks non-Normal Kubernetes events with their type (e.g. `Warning`).

```yaml
statusRules:
  - name: EventType
    start:
      domain: k8s
      classes: [Event.v1, Event.v1.events.k8s.io]
    status: |-
      {{- with index . "type"}}{{if ne . "Normal"}}{{.}}{{end}}{{end}}
```

### Kubernetes health status

Evaluates the health of any Kubernetes resource using the [kube-health](https://github.com/rhobs/kube-health) library.
Analyzes observed generation and standard Kubernetes conditions (e.g. `Ready`, `Available`, `MemoryPressure`)
to produce `Error` or `Warning` statuses. Objects without a `status` field or with healthy conditions produce no status.

```yaml
statusRules:
  - name: HealthStatus
    start:
      domain: k8s
    status: |-
      {{- k8sHealthStatus . -}}
```

### Kubernetes finalizers

Marks any Kubernetes resource that has finalizers with `Finalizer`.

```yaml
statusRules:
  - name: HasFinalizer
    start:
      domain: k8s
    status: |-
      {{- with index .metadata "finalizers"}}Finalizer{{end}}
```

## Custom status rules

Add a `statusRules` section to any rule YAML file in your [configuration](../configuration/#statusrules).
For example, to mark Pods that are not in "Running" phase:

```yaml
statusRules:
  - name: PodPhase
    start:
      domain: k8s
      classes: [Pod]
    status: |-
      {{- with .status.phase}}{{if ne . "Running"}}{{.}}{{end}}{{end}}
```

## Statuses in the API

In the [REST API](../reference/rest/) and MCP tools, statuses appear in `QueryCount` objects:

```json
{
  "query": "log:application:{kubernetes_namespace_name=\"myapp\"}",
  "count": 200,
  "statuses": [
    {"status": "Error", "count": 12},
    {"status": "Warning", "count": 45}
  ]
}
```
