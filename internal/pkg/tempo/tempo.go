package tempo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// TraceResponse represents the JSON structure of a response containing traces
type TraceResponse struct {
	Traces []Trace `json:"traces"`
}

// Trace represents a single trace in the response
type Trace struct {
	TraceID   string `json:"traceID"`
	StartTime string `json:"startTimeUnixNano"`
	//Duration  int     `json:"durationMs"`
	SpanSet SpanSet `json:"spanSet"`
	//Services  []Service `json:"services"`
}

// SpanSet represents a list of spans within a trace
type SpanSet struct {
	Match int    `json:"match"`
	Spans []Span `json:"spans"`
}

// Span represents an individual span within a service
type Span struct {
	SpanID    string `json:"spanID"`
	StartTime string `json:"startTimeUnixNano"`
	//Duration   int64      `json:"durationNanos"`
	Attributes []Attributes `json:"attributes"`
}

// Attributes map of labels associated with a spac.
type Attributes map[string]any

// CollectFunc is called for each entry returned by a query.
type CollectFunc func(*Trace)

// Client for loki HTTP API
type Client struct {
	c    *http.Client
	base *url.URL
}

func New(c *http.Client, base *url.URL) *Client { return &Client{c: c, base: base} }

// Get uses the plain Tempo API to get trace for a TraceQL query with a Constraint.
func (c *Client) Get(ctx context.Context, traceQL string, collect CollectFunc) error {
	u := c.queryURL(traceQL)
	//u := c.queryURL(traceQL, constraint)
	return c.get(ctx, u, collect)
}

// GetStack uses the TempoStack tenant API to get tracees for a TraceQL query with a Constraint.
func (c *Client) GetStack(ctx context.Context, traceQL string, constraint *korrel8r.Constraint, collect CollectFunc) error {
	u := c.queryURL(traceQL)
	//u := c.queryURL(traceQL, constraint)
	//u.Path = path.Join(tempoStackPath, tenant, u.Path)
	return c.get(ctx, u, collect)
}

const ( // Query URL keywords
	query = "q"

	//tempoStackPath = "/api/traces/v1/"
	//queryRangePath = "/tempo/api/search"
)

func (c *Client) queryURL(traceQL string) *url.URL {
	v := url.Values{}
	v.Add(query, traceQL)
	return &url.URL{RawQuery: v.Encode()}
}

func (c *Client) get(ctx context.Context, u *url.URL, collect CollectFunc) error {
	u = c.base.ResolveReference(u)
	logging.Log().V(4).Info("tempo get", "url", u)
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	qr := TraceResponse{}

	// Check for response status
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Non-OK HTTP status:", resp.StatusCode)
		return fmt.Errorf("%v", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return fmt.Errorf("%v: %v", resp.Status, string(body))
	}

	err = json.Unmarshal(body, &qr)
	if err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return err
	}

	// Process and print the traces
	for _, trace := range qr.Traces {
		fmt.Printf("TraceID: %s, StartTime: %s\n", trace.TraceID, trace.StartTime)
		// Print other relevant fields
		collect(&Trace{TraceID: trace.TraceID, StartTime: trace.StartTime})
	}
	return nil
}
