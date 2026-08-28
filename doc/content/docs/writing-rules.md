---
title: Writing Rules
description: Adding custom correlation rules
weight: 8
---

Korrel8r comes with a comprehensive set of [rules](../introduction/#rules) for correlating
Kubernetes resources and observability signals.
You can add your own rules to handle custom relationships -- for example,
correlating a custom resource with its logs or metrics.

## Rule Basics

A rule defines a relationship between _start_ classes and _goal_ classes.
It contains a template that takes a start object and generates a query for the goal class.
Both types of rule use the same YAML metadata schema to describe start and goal classes.

If a template returns a blank string or raises an error, korrel8r skips the rule for that object.

## Two Ways to Write Rules

### Configuration Rules (YAML)

Configuration rules are YAML files loaded at runtime from the [configuration](../configuration/).
They use Go `text/template` syntax and can be added or changed without rebuilding korrel8r.

See the [Configuration Rules Reference](../reference/configuration-rules/) for the full syntax,
template patterns, and examples.

### Compiled Rules (Quicktemplate)

Compiled rules are [quicktemplate](https://github.com/valyala/quicktemplate) functions
compiled into the binary as Go code.
They are faster and type-safe, but require rebuilding the executable.

See the [Compiled Rules Reference](../reference/quickrules/) for the full guide to writing,
compiling, and testing compiled rules.
