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

	"github.com/alanconway/korrel8/pkg/korrel8"
	"golang.org/x/net/html"
)

// FIXME: currently assumption of viaq vocabulary is baked in, separate store query format from domain vocabulary.

const Domain = "logs.loki-viaq"

type Class struct{}

func (c Class) Domain() korrel8.Domain { return Domain }

var _ korrel8.Class = Class{} // Implements interface.

type Object string // Log record

func (o Object) Class() korrel8.Class { return Class{} }

type Identifier Object

func (o Object) Identifier() korrel8.Identifier { return Identifier(o) }

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

// Execute LogQL query.
func (s *Store) Query(ctx context.Context, query string) (result []korrel8.Object, err error) {
	// FIXME specify limit?.
	u := *s.u
	u.Path = path.Join(u.Path, "query_range")
	q := url.Values{}
	q.Add("query", query)
	q.Add("limit", "100")
	q.Add("direction", "FORWARD")
	u.RawQuery = q.Encode()
	header := http.Header{}
	// FIXME org ID
	// if orgID != "" {
	// 	header.Add("X-Scope-OrgID", orgID)
	// }
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
	var objs []korrel8.Object
	for _, sv := range qr.Data.Result {
		for _, tl := range sv.Values { // tl is [time, line]
			objs = append(objs, Object(tl[1]))
		}
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
