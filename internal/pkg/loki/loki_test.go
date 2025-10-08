// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package loki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/stretchr/testify/assert"
)

func TestValue_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Log
		wantErr  bool
	}{
		{
			name:     "valid timestamp and line",
			input:    `["1672574400123456789", "test log line"]`,
			expected: Log{Time: time.Unix(0, 1672574400123456789), Body: "test log line"},
			wantErr:  false,
		},
		{
			name:     "epoch timestamp",
			input:    `["0", "epoch log"]`,
			expected: Log{Time: time.Unix(0, 0), Body: "epoch log"},
			wantErr:  false,
		},
		{
			name:     "large timestamp",
			input:    `["9223372036854775807", "max timestamp"]`,
			expected: Log{Time: time.Unix(0, 9223372036854775807), Body: "max timestamp"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v Log
			err := v.UnmarshalJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if v.Time.UnixNano() != tt.expected.Time.UnixNano() {
					t.Errorf("UnmarshalJSON() time = %v, want %v", v.Time, tt.expected.Time)
				}
				if v.Body != tt.expected.Body {
					t.Errorf("UnmarshalJSON() line = %v, want %v", v.Body, tt.expected.Body)
				}
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "epoch time",
			input:    time.Unix(0, 0),
			expected: "0",
		},
		{
			name:     "specific time",
			input:    time.Date(2023, 1, 1, 12, 0, 0, 123456789, time.UTC),
			expected: "1672574400123456789",
		},
		{
			name:     "time with nanoseconds",
			input:    time.Unix(1, 999999999),
			expected: "1999999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.input)
			if result != tt.expected {
				t.Errorf("formatTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestQueryURL(t *testing.T) {
	tests := []struct {
		name       string
		logQL      string
		constraint *korrel8r.Constraint
		expected   map[string]string
	}{
		{
			name:  "basic query",
			logQL: `{app="test"}`,
			constraint: &korrel8r.Constraint{
				Limit: ptr.To(100),
			},
			expected: map[string]string{
				"query": `{app="test"}`,
				"limit": "100",
			},
		},
		{
			name:  "query with time range",
			logQL: `{service="api"}`,
			constraint: &korrel8r.Constraint{
				Start: ptr.To(time.Unix(1000, 0)),
				End:   ptr.To(time.Unix(2000, 0)),
				Limit: ptr.To(50),
			},
			expected: map[string]string{
				"query": `{service="api"}`,
				"limit": "50",
				"start": "1000000000000",
				"end":   "2000000000000",
			},
		},
		{
			name:  "query with start only",
			logQL: `{container="nginx"}`,
			constraint: &korrel8r.Constraint{
				Start: ptr.To(time.Unix(1500, 0)),
			},
			expected: map[string]string{
				"query": `{container="nginx"}`,
				"start": "1500000000000",
				// end should be present (auto-added when start is set)
			},
		},
		{
			name:  "query with end only",
			logQL: `{job="prometheus"}`,
			constraint: &korrel8r.Constraint{
				End: ptr.To(time.Unix(3000, 0)),
			},
			expected: map[string]string{
				"query": `{job="prometheus"}`,
				"end":   "3000000000000",
			},
		},
		{
			name:       "minimal query",
			logQL:      `{instance="localhost"}`,
			constraint: &korrel8r.Constraint{},
			expected: map[string]string{
				"query": `{instance="localhost"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := queryURL(tt.logQL, tt.constraint)

			if u.Path != queryRangePath {
				t.Errorf("queryURL() path = %v, want %v", u.Path, queryRangePath)
			}

			values, err := url.ParseQuery(u.RawQuery)
			if err != nil {
				t.Errorf("Failed to parse query string: %v", err)
				return
			}

			for key, expectedValue := range tt.expected {
				if got := values.Get(key); got != expectedValue {
					t.Errorf("queryURL() %s = %v, want %v", key, got, expectedValue)
				}
			}
		})
	}
}

func TestCollectSorted(t *testing.T) {
	// Helper to create time from seconds
	timeFromSec := func(sec int64) time.Time {
		return time.Unix(sec, 0)
	}

	tests := []struct {
		name     string
		streams  []stream
		expected []Log
	}{
		{
			name:     "empty streams",
			streams:  []stream{},
			expected: nil,
		},
		{
			name: "single stream",
			streams: []stream{
				{
					Stream: map[string]string{"app": "test"},
					Values: []Log{
						{Time: timeFromSec(300), Body: "log3"},
						{Time: timeFromSec(200), Body: "log2"},
						{Time: timeFromSec(100), Body: "log1"},
					},
				},
			},
			expected: []Log{
				{Time: timeFromSec(300), Body: "log3", Labels: Labels{"app": "test"}},
				{Time: timeFromSec(200), Body: "log2", Labels: Labels{"app": "test"}},
				{Time: timeFromSec(100), Body: "log1", Labels: Labels{"app": "test"}},
			},
		},
		{
			name: "multiple streams sorted by timestamp",
			streams: []stream{
				{
					Stream: map[string]string{"service": "api"},
					Values: []Log{
						{Time: timeFromSec(300), Body: "api3"},
						{Time: timeFromSec(100), Body: "api1"},
					},
				},
				{
					Stream: map[string]string{"service": "web"},
					Values: []Log{
						{Time: timeFromSec(400), Body: "web4"},
						{Time: timeFromSec(200), Body: "web2"},
					},
				},
			},
			expected: []Log{
				{Time: timeFromSec(400), Body: "web4", Labels: Labels{"service": "web"}},
				{Time: timeFromSec(300), Body: "api3", Labels: Labels{"service": "api"}},
				{Time: timeFromSec(200), Body: "web2", Labels: Labels{"service": "web"}},
				{Time: timeFromSec(100), Body: "api1", Labels: Labels{"service": "api"}},
			},
		},
		{
			name: "streams with mixed empty and non-empty",
			streams: []stream{
				{
					Stream: map[string]string{"app": "empty"},
					Values: []Log{},
				},
				{
					Stream: map[string]string{"app": "test"},
					Values: []Log{
						{Time: timeFromSec(100), Body: "log1"},
						{Time: timeFromSec(200), Body: "log2"},
					},
				},
			},
			expected: []Log{
				{Time: timeFromSec(100), Body: "log1", Labels: Labels{"app": "test"}},
				{Time: timeFromSec(200), Body: "log2", Labels: Labels{"app": "test"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var collected []Log
			collectFunc := func(entry *Log) { collected = append(collected, *entry) }
			collectSorted(tt.streams, collectFunc)
			assert.Equal(t, collected, tt.expected)
		})
	}
}

func TestClient_GetError(t *testing.T) {
	tests := []struct {
		name           string
		response       interface{}
		expectedErrMsg string
	}{
		{
			name: "non-success status",
			response: response{
				Status: "error",
				Data:   data{ResultType: "streams"},
			},
			expectedErrMsg: "expected 'status: success'",
		},
		{
			name: "wrong result type",
			response: response{
				Status: "success",
				Data:   data{ResultType: "matrix"},
			},
			expectedErrMsg: "expected 'resultType: streams'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			baseURL, _ := url.Parse(server.URL)
			client := New(server.Client(), baseURL)

			collectFunc := func(entry *Log) {}
			constraint := &korrel8r.Constraint{}

			err := client.Get(context.Background(), `{app="test"}`, constraint, collectFunc)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !containsString(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error containing '%s', got: %v", tt.expectedErrMsg, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	httpClient := &http.Client{}
	baseURL, _ := url.Parse("http://example.com")

	client := New(httpClient, baseURL)

	if client.Client != httpClient {
		t.Error("Expected HTTP client to be set")
	}
	if client.BaseURL != baseURL {
		t.Error("Expected base URL to be set")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				func() bool {
					for i := 0; i <= len(s)-len(substr); i++ {
						if s[i:i+len(substr)] == substr {
							return true
						}
					}
					return false
				}())))
}
