// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package log is a domain for openshift-logging logs stored in Loki or LokiStack.
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
// A log object is a log record string in the Viaq JSON format stored by openshift logging.
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

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/openshift"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain     = Domain
	_ openshift.Converter = Domain
	_ korrel8r.Store      = &store{}
	_ korrel8r.Store      = &stackStore{}
	_ korrel8r.Query      = Query{}
	_ korrel8r.Class      = Class("")
	_ korrel8r.Previewer  = Class("")
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
	c, s, err := impl.ParseQueryString(d, s)
	if err != nil {
		return nil, err
	}
	return NewQuery(c.(Class), s), nil
}

const (
	StoreKeyLoki      = "loki"
	StoreKeyLokiStack = "lokiStack"
)

func (domain) Store(sc korrel8r.StoreConfig) (korrel8r.Store, error) {
	hc, err := k8s.NewHTTPClient()
	if err != nil {
		return nil, err
	}

	loki, lokiStack := sc[StoreKeyLoki], sc[StoreKeyLokiStack]
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

func (domain) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Add("q", q.logQL+"|json")
	v.Add("tenant", q.Class().Name())
	return &url.URL{Path: "/monitoring/logs", RawQuery: v.Encode()}, nil
}

func (domain) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	q := u.Query().Get("q")
	c := classMap[u.Query().Get("tenant")]
	if c == nil {
		c = queryClass(q)
	}
	if c == nil {
		return nil, fmt.Errorf("not a valid Loki URL: %v", u)
	}
	return NewQuery(c.(Class), q), nil
}

// Class is the log_type name (aka logType in lokistack)
type Class string

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return string(c) }
func (c Class) String() string          { return korrel8r.ClassName(c) }
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

func (c Class) New() korrel8r.Object { return Object{} }
func (c Class) Preview(x korrel8r.Object) (s string) {
	if o, ok := x.(Object); ok {
		if m := o.Properties()["message"]; m != nil {
			s, _ = m.(string)
		}
		if s == "" {
			s = o.Line()
		}
	}
	return s
}

// Object is a log record string, with on demand JSON parsing.
// Exact format depends on source of logs.
type Object struct {
	line  string
	props any
}

func NewObject(s string) Object {
	return Object{line: s}
}

// Line returns the original log line string.
func (o Object) Line() string { return o.line }

// Properties returns a the log record's property map if it has one.
func (o Object) Properties() map[string]any {
	if o.props == nil {
		if err := json.Unmarshal([]byte(o.line), &o.props); err != nil {
			o.props = err
		}
	}
	props, _ := o.props.(map[string]any)
	return props
}

// Query is a LogQL query string
type Query struct {
	logQL string // `json:",omitempty"`
	class Class  // `json:",omitempty"`
}

func NewQuery(c Class, logQL string) korrel8r.Query { return Query{class: c, logQL: logQL} }

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
func (q Query) Query() string         { return q.logQL }
func (q Query) String() string        { return korrel8r.QueryName(q) }

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
func (s *store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(q.logQL, c, func(e *loki.Entry) { result.Append(NewObject(e.Line)) })
}

type stackStore struct{ store }

func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	class := queryClass(q.logQL)
	if c == nil {
		class = Application
	}
	return s.Client.GetStack(q.logQL, class.Name(), c, func(e *loki.Entry) { result.Append(NewObject(e.Line)) })
}

var logTypeRe = regexp.MustCompile(`{[^}]*log_type(=~*)"([^"]+)"}`)

// queryClass get the class name implied by a LogQL query or nil.
func queryClass(logQL string) korrel8r.Class {
	// Parser at github.com/grafana/loki/logql does not work with go modules.
	// See https://github.com/grafana/loki/issues/2826][v2 go module semantic versioning
	// Use a simple regexp approach instead.
	if m := logTypeRe.FindStringSubmatch(logQL); m != nil {
		switch m[1] {
		case "=":
			return classMap[m[2]]
		case "=~":
			if re, err := regexp.Compile(m[2]); err == nil {
				for k, v := range classMap {
					if re.MatchString(k) {
						return v
					}
				}
			}
		}
	}
	return nil
}
