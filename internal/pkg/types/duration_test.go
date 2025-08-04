// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package types

import (
	"testing"
	"time"
)

func TestMilliDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    MilliDuration
		expected string
	}{
		{
			name:     "zero duration",
			input:    MilliDuration{Duration: 0},
			expected: "0",
		},
		{
			name:     "one millisecond",
			input:    MilliDuration{Duration: time.Millisecond},
			expected: "1",
		},
		{
			name:     "one second",
			input:    MilliDuration{Duration: time.Second},
			expected: "1000",
		},
		{
			name:     "negative duration",
			input:    MilliDuration{Duration: -time.Second},
			expected: "-1000",
		},
		{
			name:     "fractional milliseconds",
			input:    MilliDuration{Duration: 1500 * time.Microsecond},
			expected: "1", // 1.5ms truncated to 1ms
		},
		{
			name:     "large duration",
			input:    MilliDuration{Duration: 24 * time.Hour},
			expected: "86400000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", string(result), tt.expected)
			}
		})
	}
}

func TestMilliDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MilliDuration
		wantErr  bool
	}{
		{
			name:     "zero",
			input:    "0",
			expected: MilliDuration{Duration: 0},
			wantErr:  false,
		},
		{
			name:     "one millisecond",
			input:    "1",
			expected: MilliDuration{Duration: time.Millisecond},
			wantErr:  false,
		},
		{
			name:     "one second",
			input:    "1000",
			expected: MilliDuration{Duration: time.Second},
			wantErr:  false,
		},
		{
			name:     "negative duration",
			input:    "-1000",
			expected: MilliDuration{Duration: -time.Second},
			wantErr:  false,
		},
		{
			name:     "string format",
			input:    `"5000"`,
			expected: MilliDuration{Duration: 5 * time.Second},
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   `"abc"`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result MilliDuration
			err := result.UnmarshalJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result.Duration != tt.expected.Duration {
				t.Errorf("UnmarshalJSON() = %v, want %v", result.Duration, tt.expected.Duration)
			}
		})
	}
}

func TestMilliDuration_String(t *testing.T) {
	tests := []struct {
		name     string
		input    MilliDuration
		expected string
	}{
		{
			name:     "zero duration",
			input:    MilliDuration{Duration: 0},
			expected: "0",
		},
		{
			name:     "one second",
			input:    MilliDuration{Duration: time.Second},
			expected: "1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			if result != tt.expected {
				t.Errorf("String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestNanoDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    NanoDuration
		expected string
	}{
		{
			name:     "zero duration",
			input:    NanoDuration{Duration: 0},
			expected: "0",
		},
		{
			name:     "one nanosecond",
			input:    NanoDuration{Duration: time.Nanosecond},
			expected: "1",
		},
		{
			name:     "one microsecond",
			input:    NanoDuration{Duration: time.Microsecond},
			expected: "1000",
		},
		{
			name:     "one millisecond",
			input:    NanoDuration{Duration: time.Millisecond},
			expected: "1000000",
		},
		{
			name:     "one second",
			input:    NanoDuration{Duration: time.Second},
			expected: "1000000000",
		},
		{
			name:     "negative duration",
			input:    NanoDuration{Duration: -time.Second},
			expected: "-1000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalJSON() = %s, want %s", string(result), tt.expected)
			}
		})
	}
}

func TestNanoDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected NanoDuration
		wantErr  bool
	}{
		{
			name:     "zero",
			input:    "0",
			expected: NanoDuration{Duration: 0},
			wantErr:  false,
		},
		{
			name:     "one nanosecond",
			input:    "1",
			expected: NanoDuration{Duration: time.Nanosecond},
			wantErr:  false,
		},
		{
			name:     "one second",
			input:    "1000000000",
			expected: NanoDuration{Duration: time.Second},
			wantErr:  false,
		},
		{
			name:     "negative duration",
			input:    "-1000000000",
			expected: NanoDuration{Duration: -time.Second},
			wantErr:  false,
		},
		{
			name:     "string format",
			input:    `"5000000000"`,
			expected: NanoDuration{Duration: 5 * time.Second},
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   `"abc"`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result NanoDuration
			err := result.UnmarshalJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result.Duration != tt.expected.Duration {
				t.Errorf("UnmarshalJSON() = %v, want %v", result.Duration, tt.expected.Duration)
			}
		})
	}
}

func TestNanoDuration_String(t *testing.T) {
	tests := []struct {
		name     string
		input    NanoDuration
		expected string
	}{
		{
			name:     "zero duration",
			input:    NanoDuration{Duration: 0},
			expected: "0",
		},
		{
			name:     "one second",
			input:    NanoDuration{Duration: time.Second},
			expected: "1000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			if result != tt.expected {
				t.Errorf("String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestDuration_RoundTrip(t *testing.T) {
	durations := []time.Duration{
		0,
		time.Nanosecond,
		time.Microsecond,
		time.Millisecond,
		time.Second,
		time.Minute,
		time.Hour,
		-time.Second,
		24 * time.Hour,
	}

	for _, original := range durations {
		t.Run("MilliDuration round trip", func(t *testing.T) {
			md := MilliDuration{Duration: original}
			data, err := md.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			var unmarshaled MilliDuration
			err = unmarshaled.UnmarshalJSON(data)
			if err != nil {
				t.Errorf("UnmarshalJSON() error = %v", err)
				return
			}

			// Note: MilliDuration truncates to millisecond precision
			expectedMs := original.Milliseconds()
			actualMs := unmarshaled.Milliseconds()
			if actualMs != expectedMs {
				t.Errorf("MilliDuration round trip failed: original = %v ms, unmarshaled = %v ms", expectedMs, actualMs)
			}
		})

		t.Run("NanoDuration round trip", func(t *testing.T) {
			nd := NanoDuration{Duration: original}
			data, err := nd.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			var unmarshaled NanoDuration
			err = unmarshaled.UnmarshalJSON(data)
			if err != nil {
				t.Errorf("UnmarshalJSON() error = %v", err)
				return
			}

			if unmarshaled.Duration != original {
				t.Errorf("NanoDuration round trip failed: original = %v, unmarshaled = %v", original, unmarshaled.Duration)
			}
		})
	}
}
