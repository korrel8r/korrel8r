// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package trace is a korrel8r domain for OpenTelemetry traces.
//
// OpenTelemetry [traces] stored in the Grafana [Tempo] data store.
//
// # Classes
//
//	trace:span
//
// # Object
//
// Represents a [span].
// A trace is a set of spans with the same trace-id, there is no explicit class representing a trace.
//
// # Query
//
// Selector has two forms:
//   - [TraceQL] query string
//   - A list of trace IDs.
//
// A [TraceQL] query can select spans from many traces.
// Example:
//
//	trace:span:{resource.k8s.namespace.name="korrel8r"}
//
// A trace-id query is a list of hexadecimal trace IDs.
// It returns all the spans included by each trace.
// Example:
//
//	trace:span:a7880cc221e84e0d07b15993358811b7,b7880cc221e84e0d07b15993358811b7
//
// # Store
//
// The trace domain accepts an optional "tempoStack" field with a URL to connect.
//
//	stores:
//	    domain: trace
//	    tempoStack: "https://url-of-tempostack"
//
// [traces]: https://opentelemetry.io/docs/concepts/signals/traces
// [Tempo]: https://grafana.com/docs/tempo/latest/
// [span]: https://opentelemetry.io/docs/concepts/signals/traces/#spans
// [TraceQL]: https://grafana.com/docs/tempo/latest/traceql/
package trace
