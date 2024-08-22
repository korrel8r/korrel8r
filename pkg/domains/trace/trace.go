// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package trace is a domain for network observability flow events stored in Tempo or TempoStack.
//
// # Class
//
// There is a single class `trace:trace`
//
// # Object
//
// A trace object is a JSON `map[string]any` in [NetFlow] format.
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
// [TraceQL]: https://grafana.com/docs/tempo/latest/traceql
//
// [Trace]: https://docs.openshift.com/container-platform/4.16/observability/distr_tracing/distr_tracing_tempo/distr-tracing-tempo-installing.html
package trace

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/korrel8r/korrel8r/internal/pkg/tempo"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Store  = &store{}
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

	tempo, tempoStack := cs[StoreKeyTempo], cs[StoreKeyTempoStack]
	switch {

	case tempo != "" && tempoStack != "":
		return nil, fmt.Errorf("can't set both tempo and tempoStack URLs")

	case tempo != "":
		u, err := url.Parse(tempo)
		if err != nil {
			return nil, err
		}
		return NewPlainTempoStore(u, hc)

	case tempoStack != "":
		u, err := url.Parse(tempoStack)
		if err != nil {
			return nil, err
		}
		return NewTempoStackStore(u, hc)

	default:
		return nil, fmt.Errorf("must set one of tempo or tempoStack URLs")
	}
}

// There is only a single class, named "trace".
type Class struct{}

func (c Class) Domain() korrel8r.Domain                     { return Domain }
func (c Class) Name() string                                { return Domain.Name() }
func (c Class) String() string                              { return impl.ClassString(c) }
func (c Class) Description() string                         { return "A set of label:value pairs identifying a trace." }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }

// Object is a map holding trace entries
type Object *tempo.Trace

// Query is a TraceQL query string
type Query string

func NewQuery(traceQL string) korrel8r.Query { return Query(traceQL) }

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return impl.QueryString(q) }

// NewTempoStackStore returns a store that uses a TempoStack observatorium-style URLs.
func NewTempoStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{tempo.New(h, base)}}, nil
}

// NewPlainTempoStore returns a store that uses plain Tempo URLs.
func NewPlainTempoStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &store{tempo.New(h, base)}, nil
}

type store struct{ *tempo.Client }

func (store) Domain() korrel8r.Domain { return Domain }
func (s *store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(ctx, q.Data(), func(e *tempo.TraceObject) { result.Append(e) })
}

type stackStore struct{ store }

func (stackStore) Domain() korrel8r.Domain { return Domain }
func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	return s.Client.GetStack(ctx, q.Data(), c, func(e *tempo.TraceObject) { result.Append(e) })
}
