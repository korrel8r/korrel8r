// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package loki is a limited client for the Loki HTTP API: https://grafana.com/docs/loki/latest/reference/api
// Should be replaced with an official Loki client package if/when one is available.
package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

// Labels map of labels associated with a stream.
type Labels map[string]string

// Entry returned by queries includes pre-record line, time and per-stream labels.
type Entry struct {
	Line   string
	Time   time.Time
	Labels Labels
}

// CollectFunc is called for each entry returned by a query.
type CollectFunc func(*Entry)

// Client for loki HTTP API
type Client struct {
	c    *http.Client
	base *url.URL
}

func New(c *http.Client, base *url.URL) *Client { return &Client{c: c, base: base} }

// Get uses the plain Loki API to get logs for a LogQL query with a Constraint.
func (c *Client) Get(ctx context.Context, logQL string, constraint *korrel8r.Constraint, collect CollectFunc) error {
	u := c.queryURL(logQL, constraint)
	return c.get(ctx, u, collect)
}

// GetStack uses the LokiStack tenant API to get logs for a LogQL query with a Constraint.
func (c *Client) GetStack(ctx context.Context, logQL, tenant string, constraint *korrel8r.Constraint, collect CollectFunc) error {
	u := c.queryURL(logQL, constraint)
	u.Path = path.Join(lokiStackPath, tenant, u.Path)
	return c.get(ctx, u, collect)
}

const ( // Query URL keywords
	query     = "query"
	direction = "direction"
	forward   = "FORWARD"
	limit     = "limit"

	lokiStackPath  = "/api/logs/v1/"
	queryRangePath = "/loki/api/v1/query_range"
)

func (c *Client) queryURL(logQL string, constraint *korrel8r.Constraint) *url.URL {
	v := url.Values{}
	v.Add(query, logQL)
	v.Add("direction", "forward")
	if constraint != nil {
		if limit := constraint.GetLimit(); limit > 0 {
			v.Add("limit", fmt.Sprintf("%v", limit))
		}
		if constraint.Start != nil {
			v.Add("start", fmt.Sprintf("%v", constraint.Start.UnixNano()))
		}
		if constraint.End != nil {
			v.Add("end", fmt.Sprintf("%v", constraint.End.UnixNano()))
		}
	}
	return &url.URL{Path: queryRangePath, RawQuery: v.Encode()}
}

func (c *Client) get(ctx context.Context, u *url.URL, collect CollectFunc) error {
	u = c.base.ResolveReference(u)
	qr := response{}
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
func least(streams []stream) int {
	// NOTE assumes query direction is "forward"
	i := -1
	ts := func(i int) time.Time { return streams[i].Values[0].Time }
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
func collectSorted(streams []stream, collect CollectFunc) {
	for {
		i := least(streams)
		if i == -1 {
			return
		}
		v := streams[i].Values[0]
		collect(&Entry{Line: v.Line, Time: v.Time, Labels: streams[i].Stream})
		streams[i].Values = streams[i].Values[1:] // Advance the stream
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
	Values []value           `json:"values"` // [ timestamp, line ] pairs
}

type value struct {
	Time time.Time
	Line string
}

// UnmarshalJSON unmarshals Value from array [nanoUnixTime, line]
func (v *value) UnmarshalJSON(data []byte) error {
	var ss [2]string
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	n, err := strconv.ParseInt(ss[0], 10, 64)
	if err != nil {
		return err
	}
	v.Time = time.Unix(0, n)
	v.Line = ss[1]
	return nil
}
