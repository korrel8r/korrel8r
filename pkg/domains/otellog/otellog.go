// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package otellog is a domain for openshift-logging OTLP logs stored in Loki or LokiStack.
//
// # Class
//
// There are 3 classes corresponding to the 3 openshift logging log types:
//
//	otellog:application
//	otellog:infrastructure
//	otellog:audit
//
// # Object
//
// A otellog object is a JSON map[string]any in ViaQ format.
//
// # Query
//
// A query is a [LogQL] query string, prefixed by the logging class, for example:
//
//	otellog:infrastructure:{ kubernetes_namespace_name="openshift-cluster-version", kubernetes_pod_name=~".*-operator-.*" }
//
// # Store
//
// To connect to a lokiStack store use this configuration:
//
//	domain: otellog
//	lokistack: URL_OF_LOKISTACK_PROXY
//
// To connect to plain loki store use:
//
//	domain: otellog
//	loki: URL_OF_LOKI
//
// [LogQL]: https://grafana.com/docs/loki/latest/query/
package otellog

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	_ korrel8r.Store  = &store{}
	_ korrel8r.Store  = &stackStore{}
	_ korrel8r.Query  = Query{}
	_ korrel8r.Class  = Class("")
)

// Domain for otellog records produced by openshift-logging.
//
// There are several possible log store configurations:
// - Default LokiStack store on current Openshift cluster: `{}`
// - Remote LokiStack: `{ "lokiStack": "https://url-of-lokistack"}`
// - Plain Loki store: `{ "loki": "https://url-of-loki"}`
var Domain = &domain{
	impl.NewDomain("otellog", "Log Records in the otel format.", Application, Infrastructure, Audit),
}

type domain struct{ *impl.Domain }

func (d *domain) Query(s string) (korrel8r.Query, error) {
	c, s, err := impl.ParseQuery(d, s)
	if err != nil {
		return nil, err
	}
	return NewQuery(c.(Class), s), nil
}

const (
	StoreKeyLoki      = "loki"
	StoreKeyLokiStack = "lokiStack"
)

func (*domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}

	loki, lokiStack := cs[StoreKeyLoki], cs[StoreKeyLokiStack]
	switch {

	case loki != "" && lokiStack != "":
		return nil, fmt.Errorf("can't set both loki and lokiStack URLs")

	case loki != "":
		u, err := url.Parse(loki)
		if err != nil {
			return nil, err
		}
		return NewPlainLokiStore(u, hc)

	case lokiStack != "":
		u, err := url.Parse(lokiStack)
		if err != nil {
			return nil, err
		}
		return NewLokiStackStore(u, hc)

	default:
		return nil, fmt.Errorf("must set one of loki or lokiStack URLs")
	}
}

// Class is the log_type name.
type Class string

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return string(c) }
func (c Class) String() string          { return impl.ClassString(c) }

func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) {
	return impl.UnmarshalAs[Object](b)
}

func (c Class) ID(o korrel8r.Object) any {
	if log, _ := o.(*OTELLog); log != nil {
		return log.Timestamp
	}
	return nil
}

// Object represents an OpenTelemetry [log]
//
// [log]: https://opentelemetry.io/docs/concepts/signals/logs/#log-record
type Object = *OTELLog

// OTELLog is an OpenTelemetry [log], the smallest unit of work for logging.
//
// Implements the OpenTelemetry API [Spec].
//
// OTELlog: [https://opentelemetry.io/docs/concepts/signals/logs/#log-record]
type OTELLog struct {
	Body       string         `json:"body"`                 // ← Log line
	Severity   string         `json:"severityText"`         // ← Can become a label
	Timestamp  time.Time      `json:"timestamp"`            // ← Loki timestamp
	Attributes map[string]any `json:"attributes,omitempty"` // ← Becomes atreams and structured metadata
}

// Query is a LogQL query string
type Query struct {
	logQL string // `json:",omitempty"`
	class Class  // `json:",omitempty"`
}

func NewQuery(c Class, logQL string) korrel8r.Query {
	logQL = strings.TrimSpace(logQL)
	if c == "" {
		c = logQueryClass(logQL)
	}
	return Query{class: c, logQL: logQL}
}

const (
	Application    Class = "application"
	Infrastructure Class = "infrastructure"
	Audit          Class = "audit"
)

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { return q.logQL }
func (q Query) String() string        { return impl.QueryString(q) }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{Client: newClient(h, base), Store: impl.NewStore(Domain)}}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &store{Client: newClient(h, base), Store: impl.NewStore(Domain)}, nil
}

type store struct {
	*Client
	*impl.Store
}

func (s *store) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(ctx, q.Data(), constraint, func(l *OTELLog) { result.Append(l) })
}

type stackStore struct{ store }

func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.GetStack(ctx, q.Data(), q.Class().Name(), constraint, func(l *OTELLog) { result.Append(l) })
}

var logTypeRe = regexp.MustCompile(`{[^}]*log_type(=~*)"([^"]+)"}`)

// queryClass get the class name implied by a LogQL query or nil.
func logQueryClass(logQL string) Class {
	// Parser at github.com/grafana/loki/logql does not work with go modules.
	// See https://github.com/grafana/loki/issues/2826][v2 go module semantic versioning
	// Use a simple regexp approach instead.
	if m := logTypeRe.FindStringSubmatch(logQL); m != nil {
		switch m[1] {
		case "=":
			return Class(m[2])
		case "=~":
			if re, err := regexp.Compile(m[2]); err == nil {
				for _, c := range Domain.Classes() {
					if re.MatchString(c.Name()) {
						return c.(Class)
					}
				}
			}
		}
	}
	return Application
}
