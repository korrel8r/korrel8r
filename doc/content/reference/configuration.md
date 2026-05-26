---
title: Configuration
description: Config file format, stores, rules, and templates
weight: 2
---

# Configuration

Korrel8r loads configuration from a file or URL specified by the `--config` option:

```bash
korrel8r --config <file_or_url>
```

The [Korrel8r project](https://github.com/korrel8r/korrel8r) provides
[example configuration files](https://github.com/korrel8r/korrel8r/tree/main/etc/korrel8r).
You can download them or use them directly via URL.

[openshift-route.yaml](https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-route.yaml)
: Run korrel8r outside the cluster, connect to stores via routes.

[openshift-svc.yaml](https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-svc.yaml)
: Run korrel8r as an in-cluster service, connect to stores via service URLs.

The configuration is a YAML file with the following sections:

## include

Other configuration fragments to include:

```yaml
include:
  - "path_or_url"
```

## stores

Connections to data stores:

```yaml
stores:
  - domain: "domain_name"    # 1. Domain name of the store (required)
    # Domain-specific fields # 2. See Domain Reference
```

Every entry in the `stores` section has a `domain` field to identify the domain.
Other fields depend on the domain, see the [Domain Reference](../domains/).

Store fields may contain [templates](#about-templates) that expand to URLs.

**Example**: configuring a store URL from an OpenShift Route resource:

```yaml
stores:
  - domain: log
    lokiStack: >-
      {{$r := query "k8s:Route.route.openshift.io/v1:{namespace: openshift-logging, name: logging-loki}" -}}
      https://{{ (first $r).Spec.Host -}}
```

1. Get a list of routes in "openshift-logging" named "logging-loki".
2. Use the `.Spec.Host` field of the first route as the host for the store URL.

## rules

Rules to relate different classes of data:

```yaml
rules:
  - name: "rule_name"          # 1. Identifies the rule in graphs and for debugging
    start:                      # 2. Start objects must belong to one of these classes
      domain: "domain_name"
      classes:
        - "class_name"
    goal:                       # 3. Goal queries retrieve one of these classes
      domain: "domain_name"
      classes:
        - "class_name"
    result:
      query: "query_template"   # 4. Go template applied with start object as context
```

Korrel8r comes with a comprehensive set of rules by default, but you can modify them or add your own.

A rule has the following key elements:

- A set of _start_ classes. The rule can apply to objects belonging to one of these classes.
- A set of _goal_ classes. The rule can generate queries for any of these classes.
- A [Go template](#about-templates) to generate a goal query from a start object.

The query template should generate a string of the form:

```
<domain-name>:<class-name>:<query-details>
```

The _query-details_ part depends on the domain, see the [Domain Reference](../domains/).

## aliases

Short-hand alias names for groups of classes:

```yaml
aliases:
  - name: "alias_name"       # 1. Can be used wherever a class name is allowed
    domain: "domain_name"    # 2. Domain for classes in this alias
    classes:                  # 3. Classes belonging to this alias
      - "class_name"
```

## About Templates

Korrel8r rules and store configuration can include
[Go templates](https://pkg.go.dev/text/template).

> **Tip**: This is the same template syntax as the `kubectl` command with the `--output=template` option.

Korrel8r provides additional _template functions_ to simplify writing rules and configurations:

- The [Sprig](http://masterminds.github.io/sprig/) library of general purpose template functions is always available.
- Some domains (for example the `k8s` domain) provide domain-specific functions, see the [Domain Reference](../domains/).
- The following function is available for store configurations:

query
: Takes a single argument, a korrel8r query string.
  Executes the query and returns the result as a `[]any`.
  May return an error.

**Example**: Query the k8s cluster for a route, extract the "host" field:

```
{{(query "k8s:Route.route.openshift.io:{namespace: netobserv, name: loki}" | first).spec.host}}
```
