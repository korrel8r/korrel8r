// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otellog

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

type LokiResponse struct {
	Status string   `json:"status"`
	Data   lokiData `json:"data"`
}

type lokiData struct {
	ResultType string       `json:"resultType"` // Usually "streams"
	Result     []lokiStream `json:"result"`
}

type lokiStream struct {
	Stream map[string]string   `json:"stream"` // Log labels (e.g. job, app, etc.)
	Values []LokiLogEntryTuple `json:"values"` // Each entry: timestamp, line, (optional) structured metadata
}

// LokiLogEntryTuple represents one entry in the "values" array.
// It can include optional structured metadata.
type LokiLogEntryTuple [3]interface{}

// ToStructuredLogEntry converts a LokiLogEntryTuple into a strongly typed struct.
func (t LokiLogEntryTuple) ToStructuredLogEntry() (*StructuredLogEntry, error) {
	entry := &StructuredLogEntry{}

	if len(t) >= 2 {
		if ts, ok := t[0].(string); ok {
			entry.Timestamp = ts
			if ns, err := strconv.ParseInt(ts, 10, 64); err == nil {
				entry.ParsedTime = time.Unix(0, ns)
			} else {
				return nil, fmt.Errorf("invalid timestamp format: %v", err)
			}
		}
		if line, ok := t[1].(string); ok {
			entry.Line = line
		}
	}
	if len(t) == 3 {
		if metadata, ok := t[2].(map[string]interface{}); ok {
			entry.StructuredMetadata = metadata
		}
	}

	return entry, nil
}

// StructuredLogEntry is a strongly typed representation of a Loki log line with optional structured metadata.
type StructuredLogEntry struct {
	// FIXME use time.Time wrapper
	Timestamp          string                 `json:"timestamp"`
	ParsedTime         time.Time              `json:"-"`
	Line               string                 `json:"line"`
	StructuredMetadata map[string]interface{} `json:"structured_metadata,omitempty"`
}

// CollectFunc is called for each entry returned by a query.
type CollectFunc func(*OtelLog)

// Client for loki HTTP API
type Client struct {
	c    *http.Client
	base *url.URL
}

func newClient(c *http.Client, base *url.URL) *Client { return &Client{c: c, base: base} }

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
	query     = "query"
	direction = "direction"
	backward  = "BACKWARD"
	limit     = "limit"

	lokiStackPath  = "/api/logs/v1/"
	queryRangePath = "/loki/api/v1/query_range"
)

func queryURL(logQL string, c *korrel8r.Constraint) *url.URL {
	v := url.Values{}
	v.Add(query, logQL)
	v.Add(direction, backward)
	if c.GetLimit() > 0 {
		v.Add("limit", fmt.Sprintf("%v", c.GetLimit()))
	}
	start, end := c.GetStart(), c.GetEnd()
	if !end.IsZero() {
		v.Add("end", formatTime(end))
	}
	if !c.Start.IsZero() {
		v.Add("start", formatTime(start))
		if end.IsZero() { // Can't have start without end.
			v.Add("end", formatTime(time.Now()))
		}
	}
	return &url.URL{Path: queryRangePath, RawQuery: v.Encode()}
}

func formatTime(t time.Time) string { return strconv.FormatInt(t.UTC().UnixNano(), 10) }

func (c *Client) get(ctx context.Context, u *url.URL, collect CollectFunc) error {
	u = c.base.ResolveReference(u)
	qr := LokiResponse{}
	if err := impl.Get(ctx, u, c.c, &qr); err != nil {
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

// least returns index of non-empty stream with the smallest timestamp, or -1 if all are empty.
func least(streams []lokiStream) int {
	i := -1
	// Parse timestamp from each stream's last value
	ts := func(i int) time.Time {
		v := streams[i].Values
		if len(v) == 0 {
			return time.Time{}
		}
		tuple := v[len(v)-1]
		if tsStr, ok := tuple[0].(string); ok {
			if ns, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
				return time.Unix(0, ns)
			}
		}
		return time.Time{} // fallback
	}

	for j := range streams {
		if len(streams[j].Values) > 0 {
			if i < 0 || ts(j).Before(ts(i)) {
				i = j
			}
		}
	}
	return i
}

// Visit each log record in the streams in order of timestamp. Consumes the streams.
func collectSorted(streams []lokiStream, collect CollectFunc) {
	for {
		i := least(streams)
		if i == -1 {
			return
		}

		for _, raw := range streams[i].Values {
			entry, _ := raw.ToStructuredLogEntry()
			//fmt.Printf("Timestamp: %s, Log: %s, Metadata: %+v\n", entry.Timestamp, entry.Line, entry.StructuredMetadata)
			otellog := &OtelLog{
				Body:      entry.Line,
				Timestamp: entry.ParsedTime,
			}
			for k, v := range streams[i].Stream {
				otellog.Attributes[k] = v
			}
			for k, v := range entry.StructuredMetadata {
				otellog.Attributes[k] = v
			}
		}
		//collect(&Entry{Line: v.Line, Time: v.Time, Labels: streams[i].Stream})
		streams[i].Values = streams[i].Values[1:] // Advance the stream
	}
}
