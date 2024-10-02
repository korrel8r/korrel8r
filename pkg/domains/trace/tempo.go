// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package trace

import (
	"context"
	"net/http"
	"net/url"

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
	return c.get(ctx, traceQL, collect)
}

const ( // Tempo query keywords and field names
	query      = "q"
	statusAttr = "status"
)

func (c *client) get(ctx context.Context, traceQL string, collect func(*Span)) error {
	u := *c.base // Copy, don't modify base.
	u.RawQuery = url.Values{query: []string{traceQL}}.Encode()
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
			// FIXME double check interpretation of tempo RootTraceName & RootServiceName
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
