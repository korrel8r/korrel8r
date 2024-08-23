// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package tempo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// TraceResponse represents the JSON structure of a response containing traces
type TraceResponse struct {
	Traces []Trace `json:"traces"`
}

// Trace represents a single trace in the response
type Trace struct {
	TraceID string  `json:"traceID"`
	SpanSet SpanSet `json:"spanSet"`
}

// Trace represents a single trace in the response
type TraceObject struct {
	TraceID string            `json:"traceID"`
	SpanID  string            `json:"spanID"`
	Labels  map[string]string `json:"labels"`
}

// SpanSet represents a list of spans within a trace
type SpanSet struct {
	Match int    `json:"match"`
	Spans []Span `json:"spans"`
}

// Span represents an individual span within a service
type Span struct {
	SpanID     string       `json:"spanID"`
	Attributes []Attributes `json:"attributes"`
}

// Attributes map of labels associated with a spac.
type Attributes struct {
	Key   string         `json:"key"`   // Field for the key
	Value AttributeValue `json:"value"` // Field for the value (or any other type if needed)
}

// Define the original struct
type AttributeValue struct {
	StringValue string `json:"stringValue"`
}

// CollectFunc is called for each entry returned by a query.
type CollectFunc func(*TraceObject)

// Client for tempo HTTP API
type Client struct {
	c    *http.Client
	base *url.URL
}

func New(c *http.Client, base *url.URL) *Client { return &Client{c: c, base: base} }

// Get uses the plain Tempo API to get trace for a TraceQL query with a Constraint.
func (c *Client) Get(ctx context.Context, traceQL string, collect CollectFunc) error {
	u := c.queryURL(traceQL)
	return c.get(ctx, u, collect)
}

// GetStack uses the TempoStack tenant API to get tracees for a TraceQL query with a Constraint.
func (c *Client) GetStack(ctx context.Context, traceQL string, constraint *korrel8r.Constraint, collect CollectFunc) error {
	u := c.queryURL(traceQL)
	return c.get(ctx, u, collect)
}

const ( // Query URL keywords
	query = "q"
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
		for _, spans := range trace.SpanSet.Spans {
			collect(&TraceObject{TraceID: trace.TraceID, SpanID: spans.SpanID, Labels: convertLabelSetToMap(spans.Attributes)})
		}
	}
	return nil
}

func convertLabelSetToMap(m []Attributes) map[string]string {
	res := make(map[string]string, len(m))
	for _, item := range m {
		modifiedKey := strings.ReplaceAll(item.Key, ".", "_")
		res[modifiedKey] = item.Value.StringValue
	}
	return res
}
