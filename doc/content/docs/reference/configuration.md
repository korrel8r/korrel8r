---
title: Configuration
description: Config file format, stores, rules, and templates
weight: 15
---

Korrel8r loads configuration from a file or URL specified by the `--config` option or the `KORREL8R_CONFIG` environment variable.

```bash
korrel8r --config <file_or_url>
```

## Built-in configuration

The released korrel8r container image includes default configuration at `/etc/korrel8r/`,
mirroring the [`etc/korrel8r/`](https://github.com/korrel8r/korrel8r/tree/main/etc/korrel8r) directory in the source repository:

```
/etc/korrel8r/
├── openshift-route.yaml   # Out-of-cluster: connect to stores via OpenShift routes
├── openshift-svc.yaml     # In-cluster: connect to stores via service URLs
└── rules/
    └── all.yaml           #  Placeholder for additional rules
```

Korrel8r's built-in correlation and status rules are compiled into the executable from
`pkg/rules/quickrules/`; they do not need to be included from configuration. 

[openshift-route.yaml](https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-route.yaml)
: Run korrel8r outside the cluster, connect to stores via routes.

[openshift-svc.yaml](https://raw.githubusercontent.com/korrel8r/korrel8r/main/etc/korrel8r/openshift-svc.yaml)
: Run korrel8r as an in-cluster service, connect to stores via service URLs.

The default deployment uses `--config=/etc/korrel8r/openshift-svc.yaml`.

## Custom configuration

Pass a local file or URL to `--config`, or set `KORREL8R_CONFIG`. For an in-cluster deployment,
store custom configuration in a ConfigMap and mount it below `/etc/korrel8r` without replacing
that directory. See [Configuring Stores](../../configuring-stores/#apply-a-custom-configuration-in-the-cluster)
for a complete deployment example.

Custom rules can be defined directly or loaded with `include`. They supplement the compiled
built-in rules; configuration does not select or disable built-in quickrules.

The configuration file supports the following sections:

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
Other fields depend on the domain, see the [Domain Reference](domains/).

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

Runtime correlation rules use Go templates to turn start objects into goal queries. See
[Configuration Rules](configuration-rules/) for the complete schema, examples, and template
behavior. Built-in rules are compiled into the executable; this section is for additional
user-defined rules.

## statusRules

Rules that generate [status](../../statuses/) for objects in a correlation graph:

```yaml
statusRules:
  - name: "rule_name"           # 1. Identifies the rule in log output
    start:                      # 2. Start objects must belong to one of these classes
      domain: "domain_name"
      classes:                  #    Optional — omit for all classes in the domain
        - "class_name"
    status: "status_template"   # 3. Go template that outputs labels, one per line
```

See [Status](../../statuses/) for details and examples.

## aliases

Short-hand alias names for groups of classes:

```yaml
aliases:
  - name: "alias_name"       # 1. Can be used wherever a class name is allowed
    domain: "domain_name"    # 2. Domain for classes in this alias
    classes:                  # 3. Classes belonging to this alias
      - "class_name"
```

## templates

Named templates that can be reused from rule, status, or store templates:

```yaml
templates:
  - name: "template_name"       # 1. Name used to invoke the template
    template: "template_body"   # 2. Go template body
```

Named templates are invoked from rule or status templates using the standard Go template syntax:

```
{{template "template_name" <data>}}
```

The `<data>` expression becomes `.` inside the named template.
Use the [sprig](http://masterminds.github.io/sprig/) `dict` function to pass named parameters:

```yaml
templates:
  - name: myHelper
    template: 'k8s:{{.class}}:{"namespace":"{{index .labels "namespace"}}"}'

rules:
  - name: MyRule
    start: {domain: alert}
    goal: {domain: k8s, classes: [Pod]}
    result:
      query: '{{template "myHelper" (dict "labels" .Labels "class" "Pod")}}'
```

Named templates defined in any configuration file (including [included](#include) files) are available to all rules across all files.

## tuning

Limits and optimizations:

```yaml
tuning:
  requestTimeout: 1m       # 1. Timeout for incoming and outgoing requests
  sessionTimeout: 5m       # 2. Idle timeout for per-user sessions
  storeRetryInterval: 10s  # 3. Minimum time between store re-creation attempts
```

Durations use Go [duration syntax](https://pkg.go.dev/time#ParseDuration), for example `30s`, `1m`, `2h`.

`requestTimeout`
: Cancels incoming or outgoing requests that take longer than this.
  Long-lived SSE subscriptions are exempt. If omitted or 0, requests never time out.

`sessionTimeout`
: Idle timeout for sessions. In server mode each authenticated user gets a session with its own
  engine, configuration and state -- see [Security](../security/).
  If omitted or 0, sessions never time out.

`storeRetryInterval`
: Minimum time between attempts to re-create a store after an error.
  Prevents a storm of expensive re-creation (DNS lookups, API discovery) on every failed query.
  Defaults to `10s` if omitted or 0.

`unsafeSharedSession`
: Skips authentication and uses a single shared session for all requests.
  {{< callout type="warning" >}}
  This disables per-user session isolation. Use only for development or testing.
  {{< /callout >}}

## About Templates

Korrel8r rules and store configuration can include [Go templates](https://pkg.go.dev/text/template).
Korrel8r provides additional [template functions](template-functions/), domains may provide additional functions -- see the [Domain Reference](domains/)

