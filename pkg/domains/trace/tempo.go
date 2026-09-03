// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/types"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/otel"
)

var log = logging.Log()

// searchResponse is the response from the /api/search endpoint.
// searchTrace and searchSpan are for decoding the search response, converted to Span for korrel8r.
// Tempo uses unit-specific time and duration encodings, they convert to unambiguous time.Time, time.Duration in public Span.

type searchResponse struct {
	Traces []searchTrace `json:"traces"`
}

type searchTrace struct {
	TraceID         TraceID              `json:"traceID"`
	RootServiceName string               `json:"rootServiceName,omitempty"`
	RootTraceName   string               `json:"rootTraceName,omitempty"`
	Start           *types.UnixNanoTime  `json:"startTimeUnixNano,omitempty"`
	Duration        *types.MilliDuration `json:"durationMs,omitempty"`
	SpanSets        []searchSpanSet      `json:"spanSets,omitempty"`
	SpanSet         *searchSpanSet       `json:"spanSet,omitempty"` // Backwards compatibility
}

type searchSpanSet struct {
	Spans []searchSpan `json:"spans"`
}

type searchSpan struct {
	SpanID     SpanID              `json:"spanID"` // Span identifier.
	Start      types.UnixNanoTime  `json:"startTimeUnixNano"`
	Duration   types.MilliDuration `json:"durationNanos"`
	Attributes otel.KeyValueList   `json:"attributes"`
}

// tracesResponse is the response from the /api/traces/{traceID} endpoint.
// Tempo has inconsistent key names across versions:
//   - Older Tempo: "batches"
//   - Newer Tempo: "resourceSpans" (camelCase, OTLP JSON)
//
// We support both by unmarshalling into a flexible structure.
type tracesResponse struct {
	ResourceSpans []tracesResourceSpans `json:"-"` // Fallback, populated via custom unmarshal
}

func (r *tracesResponse) UnmarshalJSON(data []byte) error {
	// Try "resourceSpans" (camelCase, OTLP JSON)
	var camel struct {
		ResourceSpans []tracesResourceSpans `json:"resourceSpans"`
	}
	if err := json.Unmarshal(data, &camel); err == nil && len(camel.ResourceSpans) > 0 {
		r.ResourceSpans = camel.ResourceSpans
		return nil
	}
	// Try "batches" (older Tempo)
	var snake struct {
		Batches []tracesResourceSpans `json:"batches"`
	}
	if err := json.Unmarshal(data, &snake); err == nil && len(snake.Batches) > 0 {
		r.ResourceSpans = snake.Batches
		return nil
	}
	// Try "resource_spans" (snake_case)
	var underscore struct {
		ResourceSpans []tracesResourceSpans `json:"resource_spans"`
	}
	if err := json.Unmarshal(data, &underscore); err == nil {
		r.ResourceSpans = underscore.ResourceSpans
		return nil
	}
	return nil
}

type tracesResourceSpans struct {
	Resource   tracesResource     `json:"resource"`
	ScopeSpans []tracesScopeSpans `json:"-"`
}

func (r *tracesResourceSpans) UnmarshalJSON(data []byte) error {
	// Try camelCase first (Tempo actual response)
	var camel struct {
		Resource   tracesResource     `json:"resource"`
		ScopeSpans []tracesScopeSpans `json:"scopeSpans"`
	}
	if err := json.Unmarshal(data, &camel); err == nil && len(camel.ScopeSpans) > 0 {
		r.Resource = camel.Resource
		r.ScopeSpans = camel.ScopeSpans
		return nil
	}
	// Try snake_case (test data / older formats)
	var snake struct {
		Resource   tracesResource     `json:"resource"`
		ScopeSpans []tracesScopeSpans `json:"scope_spans"`
	}
	if err := json.Unmarshal(data, &snake); err == nil {
		r.Resource = snake.Resource
		r.ScopeSpans = snake.ScopeSpans
		return nil
	}
	return nil
}

type tracesResource struct {
	Attributes otel.KeyValueList `json:"attributes"`
}

type tracesScopeSpans struct {
	Spans []tracesSpan `json:"spans"`
}

type tracesSpan struct {
	TraceID           []byte            `json:"trace_id"`
	SpanID            []byte            `json:"span_id"`
	ParentSpanID      []byte            `json:"parent_span_id"`
	Name              string            `json:"name"`
	StartTimeUnixNano json.Number       `json:"start_time_unix_nano"`
	EndTimeUnixNano   json.Number       `json:"end_time_unix_nano"`
	Attributes        otel.KeyValueList `json:"attributes"`
	Status            tracesStatus      `json:"status"`
}

type tracesStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type client struct {
	hc   *http.Client
	base *url.URL
}

const (
	apiSearchPath = "/api/search"
	apiTracesPath = "/api/traces"
)

func newClient(c *http.Client, base *url.URL) *client {
	newBase := *base // Copy of base
	// For backwards compatibility, strip /api/search from the end of base path.
	newBase.Path = strings.TrimSuffix(newBase.Path, apiSearchPath)
	return &client{
		hc:   c,
		base: &newBase,
	}
}

// TODO: removed plain tempo store, not currently used. Put it back if necessary.
// func (c *client) Get(ctx context.Context, traceQL string, collect func(*Span)) error { ... }

// GetStack uses the TempoStack tenant API to get tracees for a TraceQL query with a Constraint.
func (c *client) GetStack(ctx context.Context, query Query, constraint *korrel8r.Constraint, collect func(*Span)) error {
	if query.isTrace() {
		return c.traces(ctx, query.Data(), constraint, collect)
	} else {
		return c.search(ctx, query.Data(), constraint, collect)
	}
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

// addConstraint adds start, end, and limit query parameters from constraint to v.
func addConstraint(v url.Values, constraint *korrel8r.Constraint) {
	if limit := constraint.GetLimit(); limit > 0 {
		v.Add("limit", strconv.Itoa(limit))
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
}

func (c *client) search(ctx context.Context, traceQL string, constraint *korrel8r.Constraint, collect func(*Span)) error {
	u := c.base.JoinPath(apiSearchPath)
	v := url.Values{query: []string{defaultSelect(traceQL)}}
	addConstraint(v, constraint)
	u.RawQuery = v.Encode()
	var response searchResponse
	if err := impl.Get(ctx, u, c.hc, &response); err != nil {
		return err
	}
	limit := constraint.GetLimit()
	count := 0
	response.collect(func(s *Span) {
		if limit > 0 && count >= limit {
			return
		}
		collect(s)
		count++
	})
	return nil
}

// traces queries the /api/traces/{traceID} endpoint to get all spans for a single trace.
func (c *client) traces(ctx context.Context, traceID string, constraint *korrel8r.Constraint, collect func(*Span)) error {
	u := c.base.JoinPath(apiTracesPath, traceID)
	v := url.Values{}
	addConstraint(v, constraint)
	if len(v) > 0 {
		u.RawQuery = v.Encode()
	}
	log.V(5).Info("traces", "url", u.String())
	var response tracesResponse
	if err := impl.Get(ctx, u, c.hc, &response); err != nil {
		return err
	}
	log.V(5).Info("traces", "resourceSpans", len(response.ResourceSpans))
	limit := constraint.GetLimit()
	count := 0
	response.collect(func(s *Span) {
		if limit > 0 && count >= limit {
			return
		}
		collect(s)
		count++
	})
	return nil
}

// collect calls collect() on each *Span.
func (r *searchResponse) collect(collect func(*Span)) {
	for _, tt := range r.Traces {
		for _, spanSet := range tt.SpanSets {
			tt.collect(spanSet, collect)
		}
		if tt.SpanSet != nil {
			tt.collect(*tt.SpanSet, collect)
		}
	}
}

// collect calls collect() on each *Span.
func (tt *searchTrace) collect(spans searchSpanSet, collect func(*Span)) {
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
		// TODO: revisit, is this correct? How does tempo represent "Ok"?
		// See otel libs for code constants.
		collect(span)
	}
}

// hexID converts a byte slice to a hex-encoded string, returning empty string for nil/empty input.
func hexID(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return fmt.Sprintf("%x", b)
}

// nanoToTime converts nanosecond Unix time (as json.Number) to time.Time.
func nanoToTime(n json.Number) time.Time {
	v, _ := n.Int64()
	return time.Unix(0, v)
}

// collect calls collect() on each *Span in a tracesResponse.
func (r *tracesResponse) collect(collect func(*Span)) {
	log.V(5).Info("traces.collect", "resourceSpans", len(r.ResourceSpans))
	for _, rs := range r.ResourceSpans {
		resourceAttrs := rs.Resource.Attributes.Map()
		log.V(5).Info("traces.collect", "scopeSpans", len(rs.ScopeSpans), "resourceAttrs", len(resourceAttrs))
		for _, ss := range rs.ScopeSpans {
			log.V(5).Info("traces.collect", "spans", len(ss.Spans))
			for _, ts := range ss.Spans {
				traceID := TraceID(hexID(ts.TraceID))
				spanID := SpanID(hexID(ts.SpanID))
				span := &Span{
					Name: ts.Name,
					Context: SpanContext{
						TraceID: traceID,
						SpanID:  spanID,
					},
					StartTime: nanoToTime(ts.StartTimeUnixNano),
					EndTime:   nanoToTime(ts.EndTimeUnixNano),
					Status:    Status{Code: StatusUnset},
				}
				if parentID := hexID(ts.ParentSpanID); parentID != "" {
					id := SpanID(parentID)
					span.ParentSpanID = &id
				}
				span.Attributes = make(map[string]any, len(resourceAttrs)+len(ts.Attributes))
				maps.Copy(span.Attributes, resourceAttrs)
				maps.Copy(span.Attributes, ts.Attributes.Map())
				if ts.Status.Message != "" {
					span.Status.Description = ts.Status.Message
				}
				switch ts.Status.Code {
				case 0: // STATUS_CODE_UNSET
					span.Status.Code = StatusUnset
				case 1: // STATUS_CODE_OK
					span.Status.Code = StatusOK
				case 2: // STATUS_CODE_ERROR
					span.Status.Code = StatusError
				}
				collect(span)
			}
		}
	}
}
