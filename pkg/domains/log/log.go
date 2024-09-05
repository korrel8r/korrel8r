// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package log is a domain for openshift-logging ViaQ logs stored in Loki or LokiStack.
//
// # Class
//
// There are 3 classes corresponding to the 3 openshift logging log types:
//
//	log:application
//	log:infrastructure
//	log:audit
//
// # Object
//
// A log object is a JSON map[string]any in ViaQ format.
//
// # Query
//
// A query is a [LogQL] query string, prefixed by the logging class, for example:
//
//	log:infrastructure:{ kubernetes_namespace_name="openshift-cluster-version", kubernetes_pod_name=~".*-operator-.*" }
//
// # Store
//
// To connect to a lokiStack store use this configuration:
//
//	domain: log
//	lokistack: URL_OF_LOKISTACK_PROXY
//
// To connect to plain loki store use:
//
//	domain: log
//	loki: URL_OF_LOKI
//
// [LogQL]: https://grafana.com/docs/loki/latest/query/
package log

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain    = Domain
	_ korrel8r.Store     = &store{}
	_ korrel8r.Store     = &stackStore{}
	_ korrel8r.Query     = Query{}
	_ korrel8r.Class     = Class("")
	_ korrel8r.Previewer = Class("")
)

// Domain for log records produced by openshift-logging.
//
// There are several possible log store configurations:
// - Default LokiStack store on current Openshift cluster: `{}`
// - Remote LokiStack: `{ "lokiStack": "https://url-of-lokistack"}`
// - Plain Loki store: `{ "loki": "https://url-of-loki"}`
var Domain = domain{}

type domain struct{}

func (domain) Name() string                     { return "log" }
func (d domain) String() string                 { return d.Name() }
func (domain) Description() string              { return "Records from container and node logs." }
func (domain) Class(name string) korrel8r.Class { return classMap[name] }
func (domain) Classes() []korrel8r.Class        { return classes }
func (d domain) Query(s string) (korrel8r.Query, error) {
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

func (domain) Store(s any) (korrel8r.Store, error) {
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

func (c Class) Domain() korrel8r.Domain                     { return Domain }
func (c Class) Name() string                                { return string(c) }
func (c Class) String() string                              { return impl.ClassString(c) }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) Preview(o korrel8r.Object) (line string)     { return Preview(o) }

// Preview extracts the message from a Viaq log record.
func Preview(x korrel8r.Object) (line string) {
	if m := x.(Object)["message"]; m != nil {
		s, _ := m.(string)
		return s
	}
	return ""
}

func (c Class) Description() string {
	switch c {
	case Application:
		return "Container logs from pods in all namespaces that do not begin with kube- or openshift-."
	case Infrastructure:
		return "Node logs (journald or syslog) and container logs from pods in namespaces that begin with kube- or openshift-."
	case Audit:
		return "Audit logs from the node operating system (/var/log/audit) and the cluster API servers"
	default:
		return ""
	}
}

// Object is a map in Viaq format.
type Object map[string]any

func NewObject(line string) Object { o, _ := impl.UnmarshalAs[Object]([]byte(line)); return o }

func (o *Object) UnmarshalJSON(line []byte) error {
	if err := json.Unmarshal([]byte(line), (*map[string]any)(o)); err != nil {
		*o = map[string]any{"message": line}
	}
	return nil
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

var (
	classes  = []korrel8r.Class{Application, Infrastructure, Audit}
	classMap = map[string]korrel8r.Class{}
)

func init() {
	for _, c := range classes {
		classMap[string(c.(Class))] = c.(Class)
	}
}

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { return q.logQL }
func (q Query) String() string        { return impl.QueryString(q) }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{loki.New(h, base)}}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &store{loki.New(h, base)}, nil
}

type store struct{ *loki.Client }

func (store) Domain() korrel8r.Domain { return Domain }

func (s *store) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(ctx, q.Data(), constraint, func(e *loki.Entry) { result.Append(NewObject(e.Line)) })
}

type stackStore struct{ store }

func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.GetStack(ctx, q.Data(), q.Class().Name(), constraint, func(e *loki.Entry) { result.Append(NewObject(e.Line)) })
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
				for _, c := range classes {
					if re.MatchString(c.Name()) {
						return c.(Class)
					}
				}
			}
		}
	}
	return Application
}
