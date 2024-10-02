// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package trace is a domain for [OpenTelemetry traces], stored in [Tempo].
//
// FIXME Re-write review all public godoc, rewrite package doc, explain OTEL relationships, link to spec.
//
// # TODO rewrite
//
// # Class
//
// There is a single class `trace:span`. A _span_ is the basic unit of work in a trace.
// A _trace_ is the set of all spans with the same `traceID`.
//
// # Object
//
// A span object is an OpenTelemetry trace in the form of a `map[string]any
//
// # Query
//
// A query is a [TraceQL] query string, prefixed by `trace:trace:`, for example:
//
//	trace:trace:{resource.k8s.namespace.name="tracing-app-k6"}
//
// # Store
//
// To connect to a tempoStack store use this configuration:
//
//	domain: trace
//	tempostack: URL_OF_TEMPOSTACK_PROXY
//
// To connect to plain tempo store use:
//
//	domain: trace
//	tempo: URL_OF_TEMPO
//
// [OpenTelemetry traces]: https://opentelemetry.io/docs/concepts/signals/traces/
// [Tempo]: https://grafana.com/docs/tempo/latest/
// [TraceQL]: https://grafana.com/docs/tempo/latest/traceql/
package trace

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Store  = &stackStore{}
	_ korrel8r.Query  = Query("")
	_ korrel8r.Class  = Class{}
)

// Domain for trace records.
//
// There are several possible trace store configurations:
// - Default TempoStack store on current Openshift cluster: `{}`
// - Remote TempoStack: `{ "tempoStack": "https://url-of-tempostack"}`
// - Plain Tempo store: `{ "tempo": "https://url-of-tempo"}`
var Domain = domain{}

type domain struct{}

func (domain) Name() string                     { return "trace" }
func (d domain) String() string                 { return d.Name() }
func (domain) Description() string              { return "Traces from Pods and Nodes." }
func (domain) Class(name string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class        { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	_, s, err := impl.ParseQuery(d, s)
	if err != nil {
		return nil, err
	}
	return Query(s), nil
}

const (
	StoreKeyTempo       = "tempo"
	StoreKeyTempoStack  = "tempoStack"
	StoreKeyTempoTenant = "tenant"
)

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	tempoStack := cs[StoreKeyTempoStack]
	if tempoStack == "" {
		return nil, fmt.Errorf("must set tempoStack URL")
	}
	u, err := url.Parse(tempoStack)
	if err != nil {
		return nil, err
	}
	return NewTempoStackStore(u, hc)
}

// There is only a single class, named "trace".
type Class struct{}

func (c Class) Domain() korrel8r.Domain                     { return Domain }
func (c Class) Name() string                                { return Domain.Name() }
func (c Class) String() string                              { return impl.ClassString(c) }
func (c Class) Description() string                         { return "A set of label:value pairs identifying a trace." }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }

// Object is a *Span.
type Object = *Span

// TraceID is a hex-encoded 16 byte identifier.
type TraceID string

// SpanID is a hex-encoded 16 byte identifier.
type SpanID string

// SpanContext identifies a span as part of a trace.
type SpanContext struct {
	TraceID TraceID `json:"traceID"`
	SpanID  SpanID  `json:"spanID"`
	// TODO TraceFlags not yet supported
}

// StatusCode FIXME spec xref
type StatusCode string

const (
	StatusUnset StatusCode = "Unset"
	StatusError StatusCode = "Error"
	StatusOK    StatusCode = "Ok"
)

// Status status of the span, see [OTEL documentation].
//
// OTEL documentation: [https://opentelemetry.io/docs/concepts/signals/traces/#span-status]
type Status struct {
	Code        StatusCode `json:"statusCode,omitempty"`  // StatusCode is "Unset", "Ok", "Error".
	Description string     `json:"description,omitempty"` // Description for status=Error.
}

// Span is an OpenTelemetry [Span], the smallest unit of work for tracing.
// Implements the OpenTelemetry API [Spec].
//
// Span: [https://opentelemetry.io/docs/concepts/signals/traces]
// Spec: [https://opentelemetry.io/docs/specs/otel/trace/api/#span]
type Span struct {
	Name       string         // Name of span.
	Context    SpanContext    `json:"context"`
	ParentID   *SpanID        `json:"spanID,omitempty"` // ParentID span ID of parent span, nil for root span.
	StartTime  time.Time      `json:"startTime"`        // StartTime for span
	EndTime    time.Time      `json:"endtime"`          // EndTime for span
	Attributes map[string]any `json:"attributes"`       // Attribute map .
	Status     Status         `json:"status"`

	// TODO OTEL links, events not yet supported.
}

// Duration is shorthand for
//
//	s.EndTime.Sub(s.StartTime)
func (s *Span) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// Query is a TraceQL query string
type Query string

func NewQuery(traceQL string) korrel8r.Query { return Query(strings.TrimSpace(traceQL)) }

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return impl.QueryString(q) }

// NewTempoStackStore returns a store that uses a TempoStack observatorium-style URLs.
func NewTempoStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{newClient(h, base)}}, nil
}

// TODO removed NewPlainTempoStore, not used initially. Restore if required.

type store struct{ *client }

func (store) Domain() korrel8r.Domain { return Domain }

type stackStore struct{ store }

func (stackStore) Domain() korrel8r.Domain { return Domain }
func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	return s.client.GetStack(ctx, q.Data(), c, func(s *Span) { result.Append(s) })
}
