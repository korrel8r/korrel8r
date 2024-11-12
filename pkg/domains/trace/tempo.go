// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package trace

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/otel"
)

// Note: tempoTrace and tempoSpan are for decoding the tempo response, converted to Span for korrel8r.
// Tempo uses unit-specific time and duration encodings, they converted to unambiguous time.Time, time.Duration in public Span.

type tempoResponse struct {
	Traces []tempoTrace `json:"traces"`
}

type tempoTrace struct {
	TraceID         TraceID            `json:"traceID"`
	RootServiceName string             `json:"rootServiceName,omitempty"`
	RootTraceName   string             `json:"rootTraceName,omitempty"`
	Start           otel.UnixNanoTime  `json:"startTimeUnixNano,omitempty"`
	Duration        otel.MilliDuration `json:"durationMs,omitempty"`
	SpanSets        []tempoSpanSet     `json:"spanSets,omitempty"`
	SpanSet         tempoSpanSet       `json:"spanSet,omitempty"` // Backwards compatibility
}

type tempoSpanSet struct {
	Spans []tempoSpan `json:"spans"`
}

type tempoSpan struct {
	SpanID     SpanID             `json:"spanID"` // Span identifier.
	Start      otel.UnixNanoTime  `json:"startTimeUnixNano"`
	Duration   otel.MilliDuration `json:"durationNanos"`
	Attributes otel.KeyValueList  `json:"attributes"`
}

type client struct {
	hc   *http.Client
	base *url.URL
}

func newClient(c *http.Client, base *url.URL) *client { return &client{hc: c, base: base} }

// TODO: removed plain tempo store, not currently used. Put it back if necessary.
// func (c *client) Get(ctx context.Context, traceQL string, collect func(*Span)) error { ... }

// GetStack uses the TempoStack tenant API to get tracees for a TraceQL query with a Constraint.
func (c *client) GetStack(ctx context.Context, traceQL string, constraint *korrel8r.Constraint, collect func(*Span)) error {
	return c.get(ctx, traceQL, constraint, collect)
}

const ( // Tempo query keywords and field names
	query      = "q"
	statusAttr = "status"
)

var (
	hasSelect         = regexp.MustCompile(`\| *select *\(`)
	defaultAttributes = strings.Join([]string{
		"resource.http.method",
		"resource.http.status_code",
		"resource.http.target",
		"resource.http.url",
		"resource.k8s.deployment.name",
		"resource.k8s.namespace.name",
		"resource.k8s.node.name",
		"resource.k8s.pod.ip",
		"resource.k8s.pod.name",
		"resource.k8s.pod.uid",
		"resource.net.host.name",
		"resource.net.host.port",
		"resource.net.peer.name",
		"resource.net.peer.port",
		"resource.service.name",
	}, ",")
)

// defaultSelect adds a default select statement to the query if there isn't one already.
func defaultSelect(traceQL string) string {
	if hasSelect.FindString(traceQL) == "" {
		return fmt.Sprintf("%v|select(%v)", traceQL, defaultAttributes)
	}
	return traceQL
}

func formatTime(t time.Time) string { return strconv.FormatInt(t.UTC().Unix(), 10) }

func (c *client) get(ctx context.Context, traceQL string, constraint *korrel8r.Constraint, collect func(*Span)) error {
	u := *c.base // Copy, don't modify base.
	v := url.Values{query: []string{defaultSelect(traceQL)}}
	if limit := constraint.GetLimit(); limit > 0 {
		v.Add("limit", strconv.Itoa(limit)) // Limit is max number of traces, not spans.
	}
	start, end := constraint.GetStart(), constraint.GetEnd()
	if !end.IsZero() {
		v.Add("end", formatTime(end))
	}
	if !start.IsZero() {
		v.Add("start", formatTime(start))
		if end.IsZero() { // Can't have start without end.
			v.Add("end", formatTime(time.Now()))
		}
	}

	u.RawQuery = v.Encode()

	var response tempoResponse
	if err := impl.Get(ctx, &u, c.hc, &response); err != nil {
		return err
	}
	response.collect(collect)
	return nil
}

// collect calls collect() on each *Span.
func (r *tempoResponse) collect(collect func(*Span)) {
	for _, tt := range r.Traces {
		for _, spanSet := range tt.SpanSets {
			tt.collect(spanSet, collect)
		}
		tt.collect(tt.SpanSet, collect)
	}
}

// collect calls collect() on each *Span.
func (tt *tempoTrace) collect(spans tempoSpanSet, collect func(*Span)) {
	for _, ts := range spans.Spans {
		span := &Span{
			Name: tt.RootTraceName,
			Context: SpanContext{
				TraceID: tt.TraceID,
				SpanID:  ts.SpanID,
			},
			StartTime: ts.Start.Time,
			EndTime:   ts.Start.Add(ts.Duration.Duration),
			Status:    Status{Code: StatusUnset}, // Default
		}
		span.Attributes = ts.Attributes.Map()
		span.Attributes[otel.AttrServiceName] = tt.RootServiceName
		// Tempo HTTP API stores span status description as "status" attribute.
		// Move it to the status field and deduce the status code.
		span.Status.Description, _ = span.Attributes[statusAttr].(string)
		delete(span.Attributes, statusAttr) // Not a real attribute.
		if span.Status.Description != "" {  // Non-empty description implies error.
			span.Status.Code = StatusError
		}
		// FIXME: revisit, is this correct? How does tempo represent "Ok"? See otel libs for code constants.
		collect(span)
	}
}
