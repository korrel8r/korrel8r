// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package trace

import (
	"encoding/json"
	"testing"
	"time"

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
		r     tempoResponse
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
