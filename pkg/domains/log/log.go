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
	"path"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"golang.org/x/exp/slices"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain    = Domain
	_ openshift.Converter  = Domain
	_ korrel8r.Store     = &Store{}
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

// TODO should be on Class?
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
	if c, ok := classMap[u.Query().Get("tenant")]; ok {
		return Query{
			class: c.(Class),
			logQL: u.Query().Get("q"),
		}, nil
	}
	return nil, fmt.Errorf("not a valid Loki URL: %v", u)
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
		classMap[string(c.(Class))] = c
	}
}

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Query() string         { return q.logQL }
func (q Query) String() string        { return korrel8r.QueryName(q) }

func (q Query) plainURL() *url.URL {
	v := url.Values{}
	v.Add("query", q.logQL)
	v.Add("direction", "forward")
	// TODO constraint inside query
	// if constraint != nil {
	// 	if constraint.Limit != nil {
	// 		v.Add("limit", fmt.Sprintf("%v", *constraint.Limit))
	// 	}
	// 	if constraint.Start != nil {
	// 		v.Add("start", fmt.Sprintf("%v", constraint.Start.UnixNano()))
	// 	}
	// 	if constraint.End != nil {
	// 		v.Add("end", fmt.Sprintf("%v", constraint.End.UnixNano()))
	// 	}
	// }
	return &url.URL{Path: "/loki/api/v1/query_range", RawQuery: v.Encode()}
}

func (q Query) lokiStackURL() *url.URL {
	u := q.plainURL()
	if q.class == "" {
		q.class = Application
	}
	u.Path = path.Join("/api/logs/v1/", q.class.Name(), u.Path)
	return u
}

type Store struct {
	c        *http.Client
	base     *url.URL
	queryURL func(Query) *url.URL
}

func (Store) Domain() korrel8r.Domain { return Domain }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, c *http.Client) (korrel8r.Store, error) {
	return &Store{c: c, base: base, queryURL: (Query).lokiStackURL}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, c *http.Client) (korrel8r.Store, error) {
	return &Store{c: c, base: base, queryURL: (Query).plainURL}, nil
}

func (s *Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	u := s.base.ResolveReference(s.queryURL(q))

	resp, err := s.c.Get(u.String())
	if err != nil {
		return fmt.Errorf("%w: %v", err, u)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%v: %v", resp.Status, u)
	}
	qr := queryResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return err
	}
	if qr.Status != "success" {
		return fmt.Errorf("expected 'status: success' in %v", qr)
	}
	if qr.Data.ResultType != "streams" {
		return fmt.Errorf("expected 'resultType: streams' in %v", qr)
	}
	// Interleave and sort the stream results.
	var logs [][]string // Each log is [timestamp,logline]
	for _, sv := range qr.Data.Result {
		logs = append(logs, sv.Values...)
	}
	slices.SortStableFunc(logs, func(a, b []string) int { return strings.Compare(a[0], b[0]) })
	for _, tl := range logs { // tl is [time, line]
		result.Append(NewObject(tl[1]))
	}
	return nil
}

// queryResponse is the response to a loki query.
type queryResponse struct {
	Status string    `json:"status"`
	Data   queryData `json:"data"`
}

// queryData holds the data for a query
type queryData struct {
	ResultType string         `json:"resultType"`
	Result     []streamValues `json:"result"`
}

// streamValues is a set of log values ["time", "line"] for a log stream.
type streamValues struct {
	Stream map[string]string `json:"stream"` // Labels for the stream
	Values [][]string        `json:"values"`
}
