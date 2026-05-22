// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package netflow is a korrel8r domain for network flow data.
//
// # Classes
//
//	netflow:network
//
// # Object
//
// JSON object in [NetFlow] format.
//
// # Query
//
// Selector is a [LogQL] query string.
// Examples:
//
//	netflow:network:{SrcK8S_Type="Pod", SrcK8S_Namespace="myNamespace"}
//	netflow:network:{DstK8S_Namespace="openshift-apiserver", DstK8S_OwnerName="apiserver"}
//
// # Store
//
// To connect to a netflow lokiStack store use this configuration:
//
//	domain: netflow
//	lokistack: URL_OF_LOKISTACK_PROXY
//
// To connect to plain loki store use:
//
//	domain: netflow
//	loki: URL_OF_LOKI
//
// [NetFlow]: https://docs.openshift.com/container-platform/latest/observability/network_observability/json-flows-format-reference.html
// [LogQL]: https://grafana.com/docs/loki/latest/query/
package netflow
