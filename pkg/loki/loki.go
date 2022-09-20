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
	"strconv"
	"strings"
	"time"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html"
)

// FIXME: currently assumption of viaq vocabulary is baked in, separate store query format from domain vocabulary.

type domain struct{}

var Domain = domain{}

func (d domain) String() string                  { return "loki" }
func (d domain) Class(name string) korrel8.Class { return Class{} }

var _ korrel8.Domain = Domain

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) String() string         { return Domain.String() }
func (c Class) New() korrel8.Object    { return Object("") }

var _ korrel8.Class = Class{} // Implements interface.

type Object string     // Log record - TODO parse as JSON Object?
type Identifier Object // The whole log record

func (o Object) Domain() korrel8.Domain { return Domain }
func (o Object) Native() any            { return o }

func (o Object) Identifier() korrel8.Identifier { return o }

var _ korrel8.Object = Object("") // Implements interface.

// Store implements the korrel8.Store interface over a Loki HTTP client
type Store struct {
	c *http.Client
	u *url.URL
}

var _ korrel8.Store = &Store{}

// NewStore creates a new store.
// baseURL is the URL with API path, e.g. "https://foo/loki/api/v1"
func NewStore(baseURL string, c *http.Client) (*Store, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Store{c: c, u: u}, nil
}

// QueryObject a JSON object representing a Loki query.
// Time values are RFC3339 format: "2006-01-02T15:04:05.999999999Z07:00"
type QueryObject struct {
	Query     string     `json:"query,omitempty"`     // LogQL log query
	Direction string     `json:"direction,omitempty"` // Direction is "FORWARD" or "BACKWARD"
	Start     *time.Time `json:"start,omitempty"`     // Start of time interval, RFC3339 format
	End       *time.Time `json:"end,omitempty"`       // End of time interval, RFC3339 format
	Limit     int        `json:"limit,omitempty"`     // Max records to retrieve
	OrgID     string     `json:"orgID,omitempty"`     // Organization ID aka tenant.
}

func (qo QueryObject) String() string {
	b, err := json.Marshal(qo)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Query executes a LogQL log query via the query_range endpoint.
//
// The query string can a JSON QueryObject or a LogQL query string.
func (s *Store) Query(ctx context.Context, query string) (result []korrel8.Object, err error) {
	qo := QueryObject{}
	d := json.NewDecoder(strings.NewReader(query))
	d.DisallowUnknownFields()
	if err := d.Decode(&qo); err != nil || qo.Query == "" { // Not a QueryObject
		qo.Query = query
	}
	u := *s.u
	u.Path = path.Join(u.Path, "query_range")
	q := url.Values{}
	q.Add("query", qo.Query)
	if qo.Limit > 0 {
		q.Add("limit", strconv.Itoa(qo.Limit))
	}
	if qo.Direction == "" {
		qo.Direction = "forward" // Change the default
	}
	if qo.Direction != "backward" && qo.Direction != "forward" {
		return nil, fmt.Errorf("Invalid direction in Loki query object: %q", qo.Direction)
	}
	q.Add("direction", qo.Direction)
	if qo.Start != nil {
		q.Add("start", fmt.Sprintf("%v", (qo.Start.UnixNano())))
	}
	if qo.End != nil {
		q.Add("end", fmt.Sprintf("%v", (qo.End.UnixNano())))
	}
	u.RawQuery = q.Encode()
	header := http.Header{}
	if qo.OrgID != "" {
		header.Add("X-Scope-OrgID", qo.OrgID)
	}
	req := &http.Request{
		Method: "GET",
		URL:    &u,
		Header: header,
	}
	resp, err := httpError(s.c.Do(req))
	if err != nil {
		return nil, fmt.Errorf("%w\nURL: %v", err, u)
	}
	defer resp.Body.Close()
	qr := queryResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return nil, err
	}
	if qr.Status != "success" {
		return nil, fmt.Errorf("expected 'status: success' in %v", qr)
	}
	if qr.Data.ResultType != "streams" {
		return nil, fmt.Errorf("expected 'resultType: streams' in %v", qr)
	}
	// FIXME return as streams??? This is inefficient.
	// Interleave and sort the stream results.
	var logs [][]string // Each log is [timestamp,logline]
	for _, sv := range qr.Data.Result {
		logs = append(logs, sv.Values...)
	}
	var less func(a, b []string) bool
	if qo.Direction == "forward" {
		less = func(a, b []string) bool { return a[0] < b[0] }
	} else {
		less = func(a, b []string) bool { return a[0] > b[0] }
	}
	slices.SortStableFunc(logs, less)
	var objs []korrel8.Object
	for _, tl := range logs { // tl is [time, line]
		objs = append(objs, Object(tl[1]))
	}
	return objs, nil
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
