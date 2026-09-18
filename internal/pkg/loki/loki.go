// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package loki is a limited client for the Loki HTTP API: https://grafana.com/docs/loki/latest/reference/api
// Should be replaced with an official Loki client package if/when one is available.
//
// NOTE: Types in this package represent the data returned by Loki.
// In particular, label names use '_' in place of '.', label values are always of type 'string'.
// Further parsing and type conversion must be done by the caller.
package loki

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/json"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/types"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var log = logging.Log()

// Log is a single loki log record.
type Log struct {
	// Body is the log line as a string.
	Body string
	// Time is the time stamp for the record.
	Time time.Time
	// Labels are the labels from the stream this record arrived on.
	Labels Labels
	// Metadata is structured metadata associated with the record.
	Metadata Labels
}

type Labels = map[string]string

// UnmarshalJSON from loki's mixed-type JSON array [time, body, metadata]
func (r *Log) UnmarshalJSON(b []byte) error {
	var tuple [3]json.RawMessage
	if err := json.Unmarshal(b, &tuple); err != nil {
		return err
	}
	if tuple[0] != nil {
		var ts types.UnixNanoTime
		if err := json.Unmarshal(tuple[0], &ts); err != nil {
			return err
		}
		r.Time = ts.Time
	}
	if tuple[1] != nil {
		if err := json.Unmarshal(tuple[1], &r.Body); err != nil {
			return err
		}
	}
	if tuple[2] != nil {
		if err := json.Unmarshal(tuple[2], &r.Metadata); err != nil {
			return err
		}
	}
	return nil
}

// CollectFunc is called for each entry returned by a query.
// The Log is valid only for the duration of the call; collectors must copy any
// data they need to retain.
type CollectFunc func(*Log)

// Client for loki HTTP API
type Client struct {
	*http.Client
	BaseURL *url.URL
}

// New loki client.
func New(c *http.Client, base *url.URL) *Client { return &Client{Client: c, BaseURL: base} }

// Get uses the plain Loki API to get logs for a LogQL query with a Constraint.
func (c *Client) Get(ctx context.Context, logQL string, constraint *korrel8r.Constraint, collect CollectFunc) error {
	u := queryURL(logQL, constraint)
	return c.get(ctx, u, collect)
}

// GetStack uses the LokiStack tenant API to get logs for a LogQL query with a Constraint.
func (c *Client) GetStack(ctx context.Context, logQL, tenant string, constraint *korrel8r.Constraint, collect CollectFunc) error {
	u := queryURL(logQL, constraint)
	u.Path = path.Join(lokiStackPath, tenant, u.Path)
	return c.get(ctx, u, collect)
}

const ( // Query URL keywords
	query = "query"
	limit = "limit"

	lokiStackPath  = "/api/logs/v1/"
	queryRangePath = "/loki/api/v1/query_range"
)

func queryURL(logQL string, c *korrel8r.Constraint) *url.URL {
	v := url.Values{}
	v.Add(query, logQL)
	if c.GetLimit() > 0 {
		v.Add(limit, fmt.Sprintf("%v", c.GetLimit()))
	}
	start, end := c.GetStart(), c.GetEnd()
	if !end.IsZero() {
		v.Add("end", formatTime(end))
	}
	if !start.IsZero() {
		v.Add("start", formatTime(start))
	}
	u := &url.URL{Path: queryRangePath, RawQuery: v.Encode()}
	return u
}

func formatTime(t time.Time) string { return strconv.FormatInt(t.UTC().UnixNano(), 10) }

func (c *Client) get(ctx context.Context, u *url.URL, collect CollectFunc) error {
	u = c.BaseURL.ResolveReference(u)
	log.V(5).Info("Loki GET", "url", u)
	qr := response{}
	if err := impl.Get(ctx, u, c.Client, &qr); err != nil {
		return err
	}
	if qr.Status != "success" {
		return fmt.Errorf("expected 'status: success' in %v", qr)
	}
	if qr.Data.ResultType != "streams" {
		return fmt.Errorf("expected 'resultType: streams' in %v", qr)
	}
	collectSorted(qr.Data.Result, collect)
	return nil
}

// Visit each log record in the streams in timestamp order.
// NOTE: assumes query direction is default "backward" (newest first).
func collectSorted(streams []stream, collect CollectFunc) {
	cursors := make(streamCursorHeap, 0, len(streams))
	for i := range streams {
		if len(streams[i].Values) > 0 {
			cursors = append(cursors, streamCursor{stream: &streams[i], streamIndex: i})
		}
	}
	cursors.init()
	for len(cursors) > 0 {
		cursor := &cursors[0]
		entry := &cursor.stream.Values[cursor.index]
		entry.Labels = cursor.stream.Stream
		collect(entry)

		// Collectors are synchronous and must copy retained data. Clear the
		// decoded entry promptly so its body and metadata can be reclaimed while
		// the remainder of a large Loki response is still being processed.
		*entry = Log{}
		cursor.index++
		if cursor.index == len(cursor.stream.Values) {
			cursors.removeRoot()
		} else {
			cursors.fixRoot()
		}
	}
}

type streamCursor struct {
	stream      *stream
	streamIndex int // Tie-break equal timestamps in original stream order.
	index       int
}

type streamCursorHeap []streamCursor

func (h streamCursorHeap) less(i, j int) bool {
	a, b := h[i], h[j]
	at, bt := a.stream.Values[a.index].Time, b.stream.Values[b.index].Time
	return at.After(bt) || (at.Equal(bt) && a.streamIndex < b.streamIndex)
}

func (h streamCursorHeap) init() {
	for i := len(h)/2 - 1; i >= 0; i-- {
		h.down(i)
	}
}

func (h streamCursorHeap) fixRoot() { h.down(0) }

func (h *streamCursorHeap) removeRoot() {
	last := len(*h) - 1
	(*h)[0] = (*h)[last]
	(*h)[last] = streamCursor{}
	*h = (*h)[:last]
	if last > 0 {
		h.down(0)
	}
}

func (h streamCursorHeap) down(parent int) {
	for {
		left := 2*parent + 1
		if left >= len(h) {
			return
		}
		best := left
		right := left + 1
		if right < len(h) && h.less(right, left) {
			best = right
		}
		if !h.less(best, parent) {
			return
		}
		h[parent], h[best] = h[best], h[parent]
		parent = best
	}
}

// Data types for query responses from  https://grafana.com/docs/loki/latest/reference/api/

type response struct {
	Status string `json:"status"`
	Data   data   `json:"data"`
}

type data struct {
	ResultType string   `json:"resultType"`
	Result     []stream `json:"result"`
}

type stream struct {
	Stream map[string]string `json:"stream"` // Labels for the stream
	Values []Log             `json:"values"` // [ timestamp, line ] pairs
}
