---
title: Writing Rules
description: Adding custom correlation rules
weight: 8
---

Korrel8r comes with a comprehensive set of [rules](../introduction/#rules-connect-data) for correlating
Kubernetes resources and observability signals.
You can add your own rules to handle custom relationships -- for example,
correlating a custom resource with its logs or metrics.

Rules are defined in YAML files and loaded from the [configuration](../configuration/).

## Rule Structure

A rule defines a relationship between a set of _start_ classes and a set of _goal_ classes.
It contains a [Go template](../configuration/#about-templates) that takes a start object
and generates a query for the goal class.

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

This rule from `etc/korrel8r/rules/k8s.yaml` finds Pods owned by resources that use label selectors
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

Korrel8r includes the [Sprig](http://masterminds.github.io/sprig/) template function library.
Some domains provide additional functions -- see the [Domain Reference](../reference/domains/)
and [Configuration Reference](../configuration/#about-templates) for details.

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

## Adding a Rule

1. Choose or create a YAML file in `etc/korrel8r/rules/`.
   Files are organized by domain: `k8s.yaml`, `log.yaml`, `alert.yaml`, etc.
   If you add a new file, include it in `all.yaml` to have it picked up by default configurations.

2. Add your rule to the `rules` list in the file.

3. If you want to [contribute your rule to the project](https://github.com/korrel8r/korrel8r), add a test case in the corresponding `*_test.go` file.
   
