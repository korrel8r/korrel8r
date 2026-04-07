// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

const Description = `

Logs can be stored on the cluster in LokiStack or in an external Loki server.
They can also be retrieved directly from the Kubernetes API server.
Direct API server access does not provide long-term log storage,
but it gives short-term access when there is no long term log store available.

## Classes

		log:application
		log:infrastructure
		log:audit

## Object

A log object is a map of attributes with string keys and values.
Attribute names contain only ASCII letters, digits, underscores, and colons (required for Loki labels).
Other characters are replaced with "_".

For stored logs: Loki stream labels and structured metadata labels become attributes.
If the log body is a JSON object, all nested field paths are flattened into attribute names.

For direct logs: the pod, namespace, and label meta-data are added as attributes
using the same label names as for stored logs.

Special attributes:

- **body**: original log message.
- **timestamp**: time the log was produced, if known, in RFC3339 format.
- **observed_timestamp**: time the log was first stored or recorded, in RFC3339 format.

### Viaq and OTEL attributes

The openshift logging collector can store logs in two formats:

- **Viaq**: legacy format with attributes like: "kubernetes_namespace_name", "kubernetes_pod_name"
- **OTEL**: new standard format with attributes like: "k8s_namespace_name", "k8s_pod_name".
  Note the use of "_" rather than "." to give legal Loki label names.
  Otherwise these names are as given in the OTEL specification.

For stored logs, korrel8r returns whatever format has been stored in Loki.

For direct logs, *both* Viaq and OTEL attributes are included to ease migration.

## Query

There are two types of query selector:
- Direct container selector: The same JSON selector used in k8s queries, with an additional "container" field.
- [LogQL](https://grafana.com/docs/loki/latest/query) expression: LogQL queries can only be used to retrieve stored logs

### Container selectors

Container selector fields (see k8s domain for more detail), all fields are optional:
- **namespace**
- **name**
- **labels**
- **fields**
- **containers**: array of container names, only get logs from these containers.

If stored logs are available, the container selector is automatically translated into
an equivalent LogQL expression.

Examples:

    log:application:{ "namespace": "something", "labels":{"app": "myapp"}, "containers":["foo", "bar"]}
    log:infrastructure:{ "namespace": "openshift-kube-apiserver", "containers":["kube-apiserver"]}


### LogQL queries

Selector is a [LogQL](https://grafana.com/docs/loki/latest/query) expression, for example:

    log:infrastructure:{kubernetes_namespace_name="openshift-cluster-version", kubernetes_pod_name=~".*-operator-.*"}

### Viaq and OTEL

When translating container selectors into LogQL, Korrel8r currently uses Viaq attributes.
This works with existing deployments and with the early support for OTEL logging,
which include backward-compatible Viaq labels.

Korrel8r will be updated to use OTEL directly in future.

## Store Configuration

    domain: log
    lokiStack: https://URL_OF_DEFAULT_LOKISTACK
    direct: true

Log queries work as follows:

1. If lokiStack is set, try to connect to the URL and retrieve stored logs.
2. If direct is true and lokiStack get fails (or lokiStack is not set): use the API server directly.

At least one of lokiStack and direct must be set.

## Template functions

The following functions can be used in rule templates when the log domain is available:

- **logTypeForNamespace**: Takes a namespace string, returns the log type for logs in that namespace; "application" or "infrastructure"
- **logSafeLabel**: Replace all characters other than alphanumerics, '_' and ':' with '_'.
- **logSafeLabels**: Takes a map[string]string argument.
  Returns a map where each key is replaced by the result of logSafeLabel
`
