// package loki generates queries for logs stored in Loki or LokiStack
//
package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type domain struct{}

func (d domain) String() string                  { return "loki" }
func (d domain) Class(name string) korrel8.Class { return classMap[name] }
func (d domain) Classes() []korrel8.Class        { return classes }

type plainRewriter struct{}

func (plainRewriter) FromQuery(q *korrel8.Query) *url.URL {
	u := *q
	u.Path = lokiStackPath.ReplaceAllString(u.Path, "")
	return &u
}
func (plainRewriter) ToQuery(u *url.URL) *url.URL {
	panic("FIXME") //Need to parse tenant from LogQL log_type?
}

type consoleRewriter struct{}

func (consoleRewriter) FromQuery(query *korrel8.Query) *url.URL {
	v := url.Values{}
	v.Add("q", query.Query().Get("query"))
	m := lokiStackPath.FindStringSubmatch(query.Path)
	if len(m) == 2 {
		v.Add("tenant", m[1])
	}
	return &url.URL{Path: "/monitoring/logs", RawQuery: v.Encode()}
}
func (consoleRewriter) ToQuery(u *url.URL) *url.URL {
	panic("FIXME") //Need to parse tenant from LogQL log_type?
}

func (d domain) URLRewriter(name string) korrel8.URLRewriter {
	switch name {
	case "console":
		return consoleRewriter{}
	case "plain":
		return plainRewriter{}
	default:
		return nil
	}
}

var Domain korrel8.Domain = domain{}

type Class string

func (c Class) Domain() korrel8.Domain   { return Domain }
func (c Class) String() string           { return string(c) }
func (c Class) New() korrel8.Object      { return Object("") }
func (c Class) Key(o korrel8.Object) any { return o }

var _ korrel8.Class = Class("") // Implements interface.

type Object string // Log record

const (
	Application    = "application"
	Infrastructure = "infrastructure"
	Audit          = "audit"
)

var (
	classNames = []string{Application, Infrastructure, Audit}
	classes    = []korrel8.Class{Class(Application), Class(Infrastructure), Class(Audit)}
	classMap   = map[string]korrel8.Class{Application: Class(Application), Infrastructure: Class(Infrastructure), Audit: Class(Audit)}
)

type Store struct {
	c    *http.Client
	base url.URL
}

var _ korrel8.Store = &Store{}

func NewStore(baseURL *url.URL, c *http.Client) (*Store, error) {
	return &Store{c: c, base: *baseURL}, nil
}

func (s *Store) String() string { return s.base.String() }

func (s *Store) Get(ctx context.Context, query *korrel8.Query, result korrel8.Result) error {
	query = s.base.ResolveReference(query)
	resp, err := httpError(s.c.Get(query.String()))
	if err != nil {
		return fmt.Errorf("%w: %v", err, query)
	}
	defer resp.Body.Close()
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
	slices.SortStableFunc(logs, func(a, b []string) bool { return a[0] < b[0] })
	for _, tl := range logs { // tl is [time, line]
		result.Append(Object(tl[1]))
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

// httpError returns err if not nil, an error constructed from resp.Body if resp.Status is not 2xx, nil otherwise.
func httpError(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, err
	}
	if resp.Status[0] == '2' {
		return resp, nil
	}
	node, _ := html.Parse(resp.Body)
	defer resp.Body.Close()
	return resp, fmt.Errorf("%v: %v", resp.Status, node.Data)
}

var lokiStackPath = regexp.MustCompile(fmt.Sprintf("/api/logs/v1/(%v)", strings.Join(classNames, "|")))

// PlainStore re-writes observatorium-style URIs as plain Loki URIs.
type PlainStore struct{ *Store }

func NewPlainStore(baseURL *url.URL, c *http.Client) (PlainStore, error) {
	s, err := NewStore(baseURL, c)
	return PlainStore{s}, err
}

func (s *PlainStore) Get(ctx context.Context, query *korrel8.Query, result korrel8.Result) error {
	u := *query
	u.Path = lokiStackPath.ReplaceAllString(u.Path, "")
	return s.Store.Get(ctx, &u, result)
}

func NewOpenshiftLokiStackStore(ctx context.Context, c client.Client, cfg *rest.Config) (*Store, error) {
	host, err := openshift.RouteHost(ctx, c, openshift.LokiStackNSName)
	if err != nil {
		return nil, err
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	return NewStore(&url.URL{Scheme: "https", Host: host}, hc)
}

func NewPlainQuery(logQL string, constraint *korrel8.Constraint) *korrel8.Query {
	v := url.Values{}
	v.Add("query", logQL)
	v.Add("direction", "forward")
	if constraint != nil {
		if constraint.Limit != nil {
			v.Add("limit", fmt.Sprintf("%v", *constraint.Limit))
		}
		if constraint.Start != nil {
			v.Add("start", fmt.Sprintf("%v", constraint.Start.UnixNano()))
		}
		if constraint.End != nil {
			v.Add("end", fmt.Sprintf("%v", constraint.End.UnixNano()))
		}
	}
	return &korrel8.Query{
		Path:     fmt.Sprintf("/loki/api/v1/query_range"),
		RawQuery: v.Encode(),
	}
}

func NewLokiStackQuery(class Class, logQL string, constraint *korrel8.Constraint) *korrel8.Query {
	u := NewPlainQuery(logQL, constraint)
	u.Path = path.Join("/api/logs/v1/", class.String(), u.Path)
	return u
}
