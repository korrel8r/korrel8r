// package loki generates queries for logs stored in Loki or LokiStack
//
package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"golang.org/x/exp/slices"
	"golang.org/x/net/html"
)

type domain struct{}

var Domain = domain{}

func (d domain) String() string                  { return "loki" }
func (d domain) Class(name string) korrel8.Class { return Class{} }
func (d domain) KnownClasses() []korrel8.Class   { return []korrel8.Class{Class{}} }

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
	c       *http.Client
	baseURL string
}

var _ korrel8.Store = &Store{}

// NewStore creates a new store.
// baseURL is the URL with API path, e.g. "https://foo/loki/api/v1"
func NewStore(baseURL string, c *http.Client) *Store {
	return &Store{c: c, baseURL: baseURL}
}

// Query executes a LogQL log query via the query_range endpoint
//
// The query string can a JSON QueryObject or a LogQL query string.
func (s *Store) Get(ctx context.Context, query string, result korrel8.Result) error {
	ustr := fmt.Sprintf("%v/%v", s.baseURL, query)
	resp, err := httpError(s.c.Get(ustr))
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
