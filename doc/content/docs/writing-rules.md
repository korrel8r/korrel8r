---
title: Writing Rules
description: Adding custom correlation rules
weight: 9
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

## Choose a rule type

| Use | Configuration rule | Compiled rule |
| --- | --- | --- |
| Intended audience | Users and operators | Korrel8r contributors |
| Source | YAML configuration | `pkg/rules/quickrules/*.qtpl` |
| Template language | Go `text/template` | Quicktemplate with type-checked Go |
| Requires rebuilding | No | Yes |
| Best for | Custom and rapidly changing relationships | Built-in, complex, or performance-sensitive relationships |

### Configuration rules

Start here when adding a relationship for your environment. Configuration rules are loaded at
runtime and can be changed without rebuilding Korrel8r. See [Configuration Rules](../reference/configuration-rules/)
for the complete schema, examples, and template functions.

### Compiled rules

Compiled rules are a contributor-facing mechanism for rules shipped in the Korrel8r executable.
They use [quicktemplate](https://github.com/valyala/quicktemplate) and require generation, compilation,
and Go tests. See [Compiled Rules](../reference/quickrules/) for the development workflow.
