---
title: Status
description: How status rules attach status to correlation graph nodes
weight: 10
---

A _status_ is an UpperCamelCase string (no spaces or punctuation) that summarizes
the "interestingness" of data in a [correlation graph](../introduction/#correlation-graphs).
A node can carry more than one status, such as `Error`, `Warning`, or `Finalizer`,
each with a count showing how many objects matched.
This lets you see which nodes have problems without retrieving the full data.

## Built-in status rules

Korrel8r ships with status rules that are compiled into the executable, for example:

- **Log severity**: mark log entries as `Error` or `Warning` based on their severity level.
- **Alert severity**: mark alerts with their severity, for example `Critical` or `Warning`.
- **Kubernetes event type**: mark events that are not of type `Normal` with their type, for example `Warning`.
- **Kubernetes health**: mark unhealthy resources as `Error` or `Warning`, based on their conditions.
- **Kubernetes finalizers**: mark resources that have finalizers with `Finalizer`.

## How status rules work

_Status rules_ in YAML [configuration](../reference/configuration/) files define how status is generated.
A status rule applies a [Go template](../reference/configuration/#about-templates) to each object retrieved during a search.
The template outputs one status per line, or nothing at all; blank lines are ignored.
Korrel8r counts how many objects produce each status and attaches the counts to the graph node.

## Custom status rules

Add a `statusRules` section to any rule YAML file in your [configuration](../reference/configuration/#statusrules).
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

The `start` field works the same as in [correlation rules](../reference/configuration-rules/#rule-structure).

Status rules can also be written as [compiled rules](../reference/quickrules/),
which is how the built-in status rules are defined.
