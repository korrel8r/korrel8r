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

Korrel8r provides additional [template functions](../reference/template-functions/) to simplify writing rules. Domains may provide additional functions -- see the [Domain Reference](../reference/domains/)

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

## Compiled Rules (Quicktemplate)

Instead of being loaded from a configuration file, rules can be compiled into the korrel8r binary
as Go code. Compiling is faster and type-safe, but requires rebuilding the executable whenever a
rule changes.

Compiled rules are [quicktemplate](https://github.com/valyala/quicktemplate) `{% func %}` templates
in `pkg/rules/quickrules/*.qtpl`. Each rule is a YAML annotation (the same schema as the
configuration rules above) followed by a template function that generates goal queries:

```text
# MetricToPod creates a k8s Pod query from the pod labels of a metric.
name: MetricToPod
start:
  domain: metric
  classes: [metric]
goal:
  domain: k8s
  classes: [Pod]

{% func MetricToPod(o interface{}) %}
{% code
	m := o.(metric.Object) // Start object, type-checked at compile time.
%}
k8s:Pod:{"namespace":{%q= m.Labels["namespace"] %},"name":{%q= m.Labels["pod"] %}}
{% endfunc %}
```

This is only an outline: `{% code %}` blocks may run arbitrary, type-checked Go. For the full
tutorial on writing, compiling, and testing quick rules, see the
[Quick Rules README](https://github.com/korrel8r/korrel8r/blob/main/pkg/rules/quickrules/README.md).

