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

	"errors"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html"
)

type domain struct{}

var Domain = domain{}

func (d domain) String() string                  { return "loki" }
func (d domain) Class(name string) korrel8.Class { return Class{} }
func (d domain) KnownClasses() []korrel8.Class   { return []korrel8.Class{Class{}} }
func (d domain) NewQuery() korrel8.Query         { var q Query; return &q }

var _ korrel8.Domain = Domain

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain         { return Domain }
func (c Class) String() string                 { return Domain.String() }
func (c Class) New() korrel8.Object            { return Object("") }
func (c Class) Contains(o korrel8.Object) bool { _, ok := o.(string); return ok }
func (c Class) Key(o korrel8.Object) any       { return o }

var _ korrel8.Class = Class{} // Implements interface.

// Query is the LogQL string
type Query string

func (q *Query) String() string { return string(*q) }

func (q *Query) REST(base *url.URL) *url.URL {
	u := *base
	u.Path = path.Join(u.Path, "query_range")
	v := url.Values{}
	v.Set("query", q.String())
	v.Set("direction", "FORWARD")
	// FIXME constraint handling
	u.RawQuery = v.Encode()
	return &u
}

func (q *Query) Browser(base *url.URL) *url.URL {
	u := *base
	u.Path = path.Join(u.Path, "monitoring/logs")
	v := url.Values{}
	v.Add("q", q.String())
	u.RawQuery = v.Encode()
	return &u
}

var _ korrel8.Query = (*Query)(nil) // Implements interface

type Object string // Log record - TODO parse as JSON Object?

func (o Object) Domain() korrel8.Domain { return Domain }

var _ korrel8.Object = Object("") // Implements interface.

// Store implements the korrel8.Store interface over a Loki HTTP client
type Store struct {
	Constraint *korrel8.Constraint
	c          *http.Client
	baseURL    url.URL
}

var _ korrel8.Store = &Store{}

// NewStore creates a new store.
// baseURL is the URL with API path, e.g. "https://foo/loki/api/v1"
func NewStore(baseURL *url.URL, c *http.Client) (*Store, error) {
	if baseURL == nil {
		return nil, errors.New("no loki URL provided")
	}
	return &Store{c: c, baseURL: *baseURL}, nil
}

// Query executes a LogQL log query via the query_range endpoint
//
// The query string can a JSON QueryObject or a LogQL query string.
func (s *Store) Get(ctx context.Context, q korrel8.Query, result korrel8.Result) error {
	query, ok := q.(*Query)
	if !ok {
		return fmt.Errorf("%v store expects %T but got %T", Domain, query, q)
	}
	resp, err := httpError(s.c.Get(query.REST(&s.baseURL).String()))
	if err != nil {
		return err
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
