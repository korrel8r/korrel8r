---
title: Configuration Rules
description: Writing YAML configuration rules with Go templates
weight: 10
---

Configuration rules are YAML files loaded at runtime from the [configuration](../../configuration/).
They use [Go template](https://pkg.go.dev/text/template) syntax to generate goal queries
from start objects. No rebuild is required to add or change a configuration rule.

## Rule Structure

A rule defines a relationship between a set of _start_ classes and a set of _goal_ classes.
It contains a Go template that takes a start object and generates a query for the goal class.

```yaml
rules:
  - name: MyRuleName
    start:
      domain: source-domain
      classes:
        - SourceClass
    goal:
      domain: target-domain
      classes:
        - TargetClass
    result:
      query: |-
        target-domain:TargetClass:query-details-template
```

- **name**: identifies the rule in graphs and log output.
- **start**: the domain and classes this rule applies to. Omitting `classes` means all classes in the domain.
- **goal**: the domain and classes the rule produces queries for.
- **result.query**: a Go template that receives a start object and outputs a goal query string.

## Example: Kubernetes Selector to Pods

This rule finds Pods owned by resources that use label selectors
(Deployments, Services, ReplicaSets, etc.):

```yaml
aliases:
  - name: selectors
    domain: k8s
    classes:
      - Service
      - Deployment.apps
      - ReplicaSet.apps
      - StatefulSet.apps

rules:
  - name: SelectorToPods
    start:
      domain: k8s
      classes: [selectors]
    goal:
      domain: k8s
      classes: [Pod]
    result:
      query: |-
        k8s:Pod:{"namespace": "{{.metadata.namespace}}"
        {{- with .spec.selector.matchLabels}}, "labels": {{mustToJson . -}}{{end -}} }
```

How it works:

1. The rule applies to any object in the `selectors` alias (Deployments, Services, etc.)
2. The template extracts the namespace and label selector from the start object.
3. It generates a `k8s:Pod` query that finds Pods matching those labels.

For example, given a Deployment in namespace `myapp` with selector `app=web`, the template produces:
```
k8s:Pod:{"namespace": "myapp", "labels": {"app":"web"}}
```

## Example: Kubernetes Resources to Metrics

This rule finds Prometheus metrics related to any Kubernetes resource:

```yaml
rules:
  - name: AllToMetric
    start:
      domain: k8s
    goal:
      domain: metric
    result:
      query: |-
        metric:metric:{namespace="{{.metadata.namespace}}",{{lower .kind}}="{{.metadata.name}}"}
```

Since `start` has no `classes` field, this rule applies to _all_ classes in the `k8s` domain.
The `lower` function converts the Kind (e.g. "Pod") to lowercase for the PromQL label name.

## Template Basics

Rule templates use [Go template syntax](https://pkg.go.dev/text/template).
The template receives the start object as its context (`.`), so you can access fields directly.

Korrel8r provides additional [template functions](../template-functions/) to simplify writing rules. Domains may provide additional functions -- see the [Domain Reference](../domains/)

Common patterns:

| Pattern | Description |
|---------|-------------|
| `{{.metadata.namespace}}` | Access a field on the start object |
| `{{.metadata.name}}` | Access another field |
| `{{with .field}}...{{end}}` | Conditionally include a section if the field exists |
| `{{range .items}}...{{end}}` | Iterate over a list |
| `{{mustToJson .field}}` | Convert a value to JSON |
| `{{lower .kind}}` | Convert to lowercase |

If a template returns a blank string or raises an error, korrel8r skips the rule for that object.
Errors are logged, blanks are ignored silently.
