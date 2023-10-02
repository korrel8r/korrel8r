// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package log is a domain for openshift-logging logs stored in Loki or LokiStack
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
	"github.com/korrel8r/korrel8r/pkg/openshift/console"
	"golang.org/x/exp/slices"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain    = Domain
	_ console.Converter  = Domain
	_ korrel8r.Store     = &Store{}
	_ korrel8r.Query     = &Query{}
	_ korrel8r.Class     = Class("")
	_ korrel8r.Previewer = Class("")
)

var Domain = domain{}

type domain struct{}

func (domain) Name() string                           { return "log" }
func (d domain) String() string                       { return d.Name() }
func (domain) Description() string                    { return "Records from container and node logs." }
func (domain) Class(name string) korrel8r.Class       { return classMap[name] }
func (domain) Classes() []korrel8r.Class              { return classes }
func (domain) Query(s string) (korrel8r.Query, error) { return impl.Query(s, &Query{}) }

const (
	StoreKeyLoki      = "loki"
	StoreKeyLokiStack = "lokiStack"
)

func (domain) Store(sc korrel8r.StoreConfig) (korrel8r.Store, error) {
	loki, lokiStack := sc[StoreKeyLoki], sc[StoreKeyLokiStack]
	if loki != "" && lokiStack != "" {
		return nil, fmt.Errorf("can't create a store with both loki and lokiStack URLs")
	}
	if loki == "" && lokiStack == "" {
		c, cfg, err := k8s.NewClient()
		if err != nil {
			return nil, err
		}
		return NewOpenshiftLokiStackStore(context.Background(), c, cfg)
	}
	s := loki
	if s == "" {
		s = lokiStack
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if lokiStack != "" {
		return NewLokiStackStore(u, nil)
	}
	return NewPlainLokiStore(u, nil)
}

func (domain) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Add("q", q.LogQL)
	v.Add("tenant", q.LogType)
	return &url.URL{Path: "/monitoring/logs", RawQuery: v.Encode()}, nil
}

func (domain) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	if c, ok := classMap[u.Query().Get("tenant")]; ok {
		return &Query{
			LogQL:   u.Query().Get("q"),
			LogType: c.Name(),
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
func (c Class) Preview(o korrel8r.Object) string {
	var s string
	if o, ok := o.(Object); ok {
		s, _ = o.JSON()["message"].(string)
		if s == "" {
			s = o.Entry
		}
	}
	return s
}

type Object struct {
	Entry string          // Log entry as a string.
	json  *map[string]any // Decoded as a JSON Object. Empty if failed.
}

func NewObject(entry string) Object { return Object{Entry: entry} }

func (o Object) JSON() map[string]any {
	if o.json == nil {
		o.json = new(map[string]any)
		_ = json.Unmarshal([]byte(o.Entry), o.json)
	}
	return *o.json
}

// Query is a LogQL query string
type Query struct {
	LogQL   string // `json:",omitempty"`
	LogType string // `json:",omitempty"`
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
		classMap[string(c.(Class))] = c
	}
}

func (q *Query) Class() korrel8r.Class { return Class(q.LogType) }
func (q *Query) String() string        { return korrel8r.JSONString(q) }

func (q *Query) plainURL() *url.URL {
	v := url.Values{}
	v.Add("query", q.LogQL)
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

func (q *Query) lokiStackURL() *url.URL {
	u := q.plainURL()
	if q.LogType == "" {
		q.LogType = Application.Name()
	}
	u.Path = path.Join("/api/logs/v1/", q.LogType, u.Path)
	return u
}

type Store struct {
	c        *http.Client
	base     *url.URL
	queryURL func(*Query) *url.URL
}

func (Store) Domain() korrel8r.Domain { return Domain }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, c *http.Client) (korrel8r.Store, error) {
	return &Store{c: c, base: base, queryURL: (*Query).lokiStackURL}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, c *http.Client) (korrel8r.Store, error) {
	return &Store{c: c, base: base, queryURL: (*Query).plainURL}, nil
}

func (s *Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
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

func NewOpenshiftLokiStackStore(ctx context.Context, c client.Client, cfg *rest.Config) (korrel8r.Store, error) {
	host, err := openshift.RouteHost(ctx, c, openshift.LokiStackNSName)
	if err != nil {
		return nil, err
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	return NewLokiStackStore(&url.URL{Scheme: "https", Host: host}, hc)
}
