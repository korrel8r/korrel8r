// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package trace

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	const response = `{
      "traces": [
        {
          "traceID": "2f3e0cee77ae5dc9c17ade3689eb2e54",
          "rootServiceName": "shop-backend",
          "rootTraceName": "update-billing",
          "startTimeUnixNano": "1684778327699392724",
          "durationMs": 557,
          "spanSets": [
            {
              "spans": [
                {
                  "spanID": "1",
                  "startTimeUnixNano": "1",
                  "durationNanos": "1",
                  "attributes": [
                    {
                      "key": "status",
                      "value": {
                        "stringValue": "something went wrong"
                      }
                    }
									]
                }
              ],
              "matched": 1
            },
            {
              "spans": [
                {
                  "spanID": "563d623c76514f8",
                  "startTimeUnixNano": "1684778327735077899",
                  "durationNanos": "44697949",
                  "attributes": [
                    {
                      "key": "answer.int",
                      "value": {
                        "intValue": 42
                      }
                    },
                    {
                      "key": "answer.float",
                      "value": {
                        "doubleValue": "42"
                      }
                    }
                  ]
                }
              ],
              "matched": 1
            }
          ],
          "spanSet": {
            "spans": [
              {
                "spanID": "563d623c76514f8e",
                "startTimeUnixNano": "2684778327735077898",
                "durationNanos": "546979497"
              }
            ],
            "matched": 1
          }
        }
      ]
}`
	var (
		r     searchResponse
		spans []*Span
	)
	require.NoError(t, json.Unmarshal([]byte(response), &r))
	r.collect(func(s *Span) { spans = append(spans, s) })
	require.NotEmpty(t, spans)
	traceID := TraceID("2f3e0cee77ae5dc9c17ade3689eb2e54")
	want := []*Span{
		{
			Name:      "update-billing",
			Context:   SpanContext{TraceID: traceID, SpanID: "1"},
			StartTime: time.Unix(0, 1),
			EndTime:   time.Unix(0, 1).Add(time.Millisecond),
			Attributes: map[string]any{
				"service.name": "shop-backend",
			},
			Status: Status{Code: StatusError, Description: "something went wrong"},
		},
		{
			Name:      "update-billing",
			Context:   SpanContext{TraceID: traceID, SpanID: "563d623c76514f8"},
			StartTime: time.Unix(0, 1684778327735077899),
			EndTime:   time.Unix(0, 1684778327735077899).Add(44697949 * time.Millisecond),
			Attributes: map[string]any{
				"service.name": "shop-backend",
				"answer.int":   int64(42),
				"answer.float": float64(42),
			},
			Status: Status{Code: StatusUnset}},
		{
			Name:      "update-billing",
			Context:   SpanContext{TraceID: traceID, SpanID: "563d623c76514f8e"},
			StartTime: time.Unix(0, 2684778327735077898),
			EndTime:   time.Unix(0, 2684778327735077898).Add(546979497 * time.Millisecond),
			Attributes: map[string]any{
				"service.name": "shop-backend",
			},
			Status: Status{Code: StatusUnset}},
	}
	assert.Equal(t, want, spans)
}

func TestTracesResponseCollect(t *testing.T) {
	const response = `{
      "resource_spans": [
        {
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "my-service"}},
              {"key": "service.namespace", "value": {"stringValue": "production"}}
            ]
          },
          "scope_spans": [
            {
              "spans": [
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "AQIDBAUGBwg=",
                  "name": "GET /api/users",
                  "start_time_unix_nano": 1684778327699392724,
                  "end_time_unix_nano": 1684778327735077898,
                  "attributes": [
                    {"key": "http.method", "value": {"stringValue": "GET"}},
                    {"key": "http.status_code", "value": {"intValue": 200}}
                  ],
                  "status": {"code": 1}
                },
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "qrvM3e7/ABE=",
                  "name": "POST /api/orders",
                  "start_time_unix_nano": 1684778327735077898,
                  "end_time_unix_nano": 1684778327800000000,
                  "attributes": [
                    {"key": "http.method", "value": {"stringValue": "POST"}},
                    {"key": "service.name", "value": {"stringValue": "override-service"}}
                  ],
                  "status": {"code": 2, "message": "internal error"}
                }
              ]
            }
          ]
        }
      ]
    }`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	require.Len(t, spans, 2)

	// First span: resource attrs merged, status OK.
	assert.Equal(t, TraceID("2f3e0cee77ae5dc9c17ade3689eb2e54"), spans[0].Context.TraceID)
	assert.Equal(t, SpanID("0102030405060708"), spans[0].Context.SpanID)
	assert.Equal(t, "GET /api/users", spans[0].Name)
	assert.Equal(t, time.Unix(0, 1684778327699392724), spans[0].StartTime)
	assert.Equal(t, time.Unix(0, 1684778327735077898), spans[0].EndTime)
	assert.Equal(t, Status{Code: StatusOK}, spans[0].Status)
	assert.Equal(t, map[string]any{
		"service.name":      "my-service",
		"service.namespace": "production",
		"http.method":       "GET",
		"http.status_code":  int64(200),
	}, spans[0].Attributes)

	// Second span: span attr overrides resource attr, status Error.
	assert.Equal(t, SpanID("aabbccddeeff0011"), spans[1].Context.SpanID)
	assert.Equal(t, "POST /api/orders", spans[1].Name)
	assert.Equal(t, Status{Code: StatusError, Description: "internal error"}, spans[1].Status)
	assert.Equal(t, "override-service", spans[1].Attributes["service.name"])
	assert.Equal(t, "POST", spans[1].Attributes["http.method"])
	// Resource attrs still present.
	assert.Equal(t, "production", spans[1].Attributes["service.namespace"])
}

func TestTracesResponseBatchesFormat(t *testing.T) {
	// Test "batches" format used by older Tempo versions
	const response = `{
      "batches": [
        {
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "my-service"}}
            ]
          },
          "scope_spans": [
            {
              "spans": [
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "AQIDBAUGBwg=",
                  "name": "GET /api/users",
                  "start_time_unix_nano": 1684778327699392724,
                  "end_time_unix_nano": 1684778327735077898,
                  "status": {"code": 1}
                }
              ]
            }
          ]
        }
      ]
    }`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	require.Len(t, spans, 1)
	assert.Equal(t, TraceID("2f3e0cee77ae5dc9c17ade3689eb2e54"), spans[0].Context.TraceID)
	assert.Equal(t, SpanID("0102030405060708"), spans[0].Context.SpanID)
	assert.Equal(t, "GET /api/users", spans[0].Name)
	assert.Equal(t, Status{Code: StatusOK}, spans[0].Status)
}

func TestTracesResponseCamelCaseFormat(t *testing.T) {
	// Test "resourceSpans" with "scopeSpans" (camelCase) - actual Tempo format
	const response = `{
      "resourceSpans": [
        {
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "my-service"}}
            ]
          },
          "scopeSpans": [
            {
              "spans": [
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "AQIDBAUGBwg=",
                  "name": "GET /api/users",
                  "start_time_unix_nano": 1684778327699392724,
                  "end_time_unix_nano": 1684778327735077898,
                  "status": {"code": 1}
                }
              ]
            }
          ]
        }
      ]
    }`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	require.Len(t, spans, 1)
	assert.Equal(t, TraceID("2f3e0cee77ae5dc9c17ade3689eb2e54"), spans[0].Context.TraceID)
	assert.Equal(t, SpanID("0102030405060708"), spans[0].Context.SpanID)
	assert.Equal(t, "GET /api/users", spans[0].Name)
	assert.Equal(t, Status{Code: StatusOK}, spans[0].Status)
}

func TestTracesResponseEmpty(t *testing.T) {
	// Test empty response
	const response = `{"resourceSpans": []}`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	assert.Empty(t, spans)
}

func TestTracesResponseNoSpans(t *testing.T) {
	// Test response with resource but no spans
	const response = `{
      "resourceSpans": [
        {
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "my-service"}}
            ]
          },
          "scopeSpans": [
            {
              "spans": []
            }
          ]
        }
      ]
    }`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	assert.Empty(t, spans)
}

func TestTracesResponseParentSpanID(t *testing.T) {
	// Test that parentSpanID is correctly parsed
	// CgoKCgoKCgoKCg== is base64 for bytes [0x0a x10] -> hex "0a0a0a0a0a0a0a0a0a0a"
	const response = `{
      "resourceSpans": [
        {
          "resource": {
            "attributes": []
          },
          "scopeSpans": [
            {
              "spans": [
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "AQIDBAUGBwg=",
                  "parent_span_id": "CgoKCgoKCgoKCg==",
                  "name": "child-span",
                  "start_time_unix_nano": 1684778327699392724,
                  "end_time_unix_nano": 1684778327735077898,
                  "status": {"code": 1}
                }
              ]
            }
          ]
        }
      ]
    }`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	require.Len(t, spans, 1)
	require.NotNil(t, spans[0].ParentSpanID)
	assert.Equal(t, SpanID("0a0a0a0a0a0a0a0a0a0a"), *spans[0].ParentSpanID)
}

func TestTracesResponseMultipleResourceSpans(t *testing.T) {
	// Test multiple resource spans
	const response = `{
      "resourceSpans": [
        {
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "service-1"}}
            ]
          },
          "scopeSpans": [
            {
              "spans": [
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "AQIDBAUGBwg=",
                  "name": "span-1",
                  "start_time_unix_nano": 1684778327699392724,
                  "end_time_unix_nano": 1684778327735077898,
                  "status": {"code": 1}
                }
              ]
            }
          ]
        },
        {
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "service-2"}}
            ]
          },
          "scopeSpans": [
            {
              "spans": [
                {
                  "trace_id": "Lz4M7neuXcnBet42iesuVA==",
                  "span_id": "qrvM3e7/ABE=",
                  "name": "span-2",
                  "start_time_unix_nano": 1684778327735077898,
                  "end_time_unix_nano": 1684778327800000000,
                  "status": {"code": 2}
                }
              ]
            }
          ]
        }
      ]
    }`
	var r tracesResponse
	require.NoError(t, json.Unmarshal([]byte(response), &r))

	var spans []*Span
	r.collect(func(s *Span) { spans = append(spans, s) })
	require.Len(t, spans, 2)
	assert.Equal(t, "span-1", spans[0].Name)
	assert.Equal(t, "service-1", spans[0].Attributes["service.name"])
	assert.Equal(t, "span-2", spans[1].Name)
	assert.Equal(t, "service-2", spans[1].Attributes["service.name"])
}

func TestAddConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint *korrel8r.Constraint
		wantKeys   []string
	}{
		{
			name:       "nil constraint",
			constraint: nil,
			wantKeys:   nil,
		},
		{
			name:       "empty constraint",
			constraint: &korrel8r.Constraint{},
			wantKeys:   nil,
		},
		{
			name: "with limit",
			constraint: &korrel8r.Constraint{
				Limit: intPtr(100),
			},
			wantKeys: []string{"limit"},
		},
		{
			name: "with start and end",
			constraint: &korrel8r.Constraint{
				Start: timePtr(time.Unix(1000, 0)),
				End:   timePtr(time.Unix(2000, 0)),
			},
			wantKeys: []string{"start", "end"},
		},
		{
			name: "with start only",
			constraint: &korrel8r.Constraint{
				Start: timePtr(time.Unix(1000, 0)),
			},
			wantKeys: []string{"start", "end"},
		},
		{
			name: "with all fields",
			constraint: &korrel8r.Constraint{
				Limit: intPtr(50),
				Start: timePtr(time.Unix(1000, 0)),
				End:   timePtr(time.Unix(2000, 0)),
			},
			wantKeys: []string{"limit", "start", "end"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := url.Values{}
			addConstraint(v, tt.constraint)
			for _, key := range tt.wantKeys {
				assert.True(t, v.Has(key), "expected key %q", key)
			}
			assert.Equal(t, len(tt.wantKeys), len(v))
		})
	}
}

func intPtr(i int) *int       { return &i }
func timePtr(t time.Time) *time.Time { return &t }

func TestHexID(t *testing.T) {
	assert.Equal(t, "", hexID(nil))
	assert.Equal(t, "", hexID([]byte{}))
	assert.Equal(t, "0102030405060708", hexID([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	assert.Equal(t, "abcdef", hexID([]byte{0xab, 0xcd, 0xef}))
}

func TestNanoToTime(t *testing.T) {
	assert.Equal(t, time.Unix(0, 0), nanoToTime(json.Number("0")))
	assert.Equal(t, time.Unix(0, 1684778327699392724), nanoToTime(json.Number("1684778327699392724")))
}
