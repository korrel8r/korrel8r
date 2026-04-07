// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package trace implements OpenTelemetry traces stored in the Grafana Tempo data store.
//
// See [Description] for details.
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

var Domain = domain{Domain: impl.NewDomain("trace", "Follow the path of a request through your application.", Description, Class{})}

type domain struct{ *impl.Domain }

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

type Class struct{}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return "span" }
func (c Class) String() string          { return korrel8r.ClassString(c) }

func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) ID(o korrel8r.Object) any {
	if span, _ := o.(Object); span != nil {
		return span.Context
	}
	return nil
}

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

// StatusCode see [Status]
type StatusCode string

const (
	StatusUnset StatusCode = "Unset"
	StatusError StatusCode = "Error"
	StatusOK    StatusCode = "Ok"
)

// Status of a span, see [OTEL documentation].
//
// OTEL documentation: [https://opentelemetry.io/docs/concepts/signals/traces/#span-status]
type Status struct {
	Code        StatusCode `json:"statusCode,omitempty"`  // StatusCode is "Unset", "Ok", "Error".
	Description string     `json:"description,omitempty"` // Description for status=Error.
}

// Span is an OpenTelemetry [span], the smallest unit of work for tracing.
//
// Implements the OpenTelemetry API [Spec].
//
// Span: [https://opentelemetry.io/docs/concepts/signals/traces]
type Span struct {
	Name       string         `json:"name"`             // Name of span.
	Context    SpanContext    `json:"context"`          // Context identifying the span.
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

type Query string

func NewQuery(traceQL string) korrel8r.Query { return Query(strings.TrimSpace(traceQL)) }

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return korrel8r.QueryString(q) }

// NewTempoStackStore returns a store that uses a TempoStack observatorium-style URLs.
func NewTempoStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{client: newClient(h, base)}}, nil
}

// TODO removed NewPlainTempoStore, not used initially. Restore if required.

type store struct {
	*client
	*impl.Store
}

func (store) Domain() korrel8r.Domain { return Domain }

type stackStore struct{ store }

func (stackStore) Domain() korrel8r.Domain { return Domain }
func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	return s.GetStack(ctx, q.Data(), c, func(s *Span) { result.Append(s) })
}
