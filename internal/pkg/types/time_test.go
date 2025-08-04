// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package types

import (
	"testing"
	"time"
)

func TestUnixNanoTime_MarshalJSON(t *testing.T) {
	// Create test times
	epoch := time.Unix(0, 0)
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 123456789, time.UTC)

	tests := []struct {
		name     string
		input    UnixNanoTime
		expected string
	}{
		{
			name:     "epoch time",
			input:    UnixNanoTime{Time: epoch},
			expected: "0",
		},
		{
			name:     "specific time",
			input:    UnixNanoTime{Time: testTime},
			expected: `"1672574400123456789"`, // As string because > 32-bit
		},
		{
			name:     "time with nanoseconds",
			input:    UnixNanoTime{Time: time.Unix(1, 999999999)},
			expected: "1999999999", // As number because < 32-bit limit
		},
		{
			name:     "small positive time",
			input:    UnixNanoTime{Time: time.Unix(0, 1000000000)}, // 1 second after epoch
			expected: "1000000000",                                 // As number because < 32-bit limit
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

func TestUnixNanoTime_UnmarshalJSON(t *testing.T) {
	epoch := time.Unix(0, 0)
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 123456789, time.UTC)

	tests := []struct {
		name     string
		input    string
		expected UnixNanoTime
		wantErr  bool
	}{
		{
			name:     "epoch time",
			input:    "0",
			expected: UnixNanoTime{Time: epoch},
			wantErr:  false,
		},
		{
			name:     "number format",
			input:    "1000000000",
			expected: UnixNanoTime{Time: time.Unix(1, 0)},
			wantErr:  false,
		},
		{
			name:     "string format",
			input:    `"1672574400123456789"`,
			expected: UnixNanoTime{Time: testTime},
			wantErr:  false,
		},
		{
			name:     "negative time",
			input:    "-1000000000",
			expected: UnixNanoTime{Time: time.Unix(-1, 0)},
			wantErr:  false,
		},
		{
			name:     "large time as string",
			input:    `"9223372036854775807"`, // MaxInt64
			expected: UnixNanoTime{Time: time.Unix(0, 9223372036854775807)},
			wantErr:  false,
		},
		{
			name:    "invalid string",
			input:   `"abc"`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `{`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   `""`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result UnixNanoTime
			err := result.UnmarshalJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Compare Unix nanoseconds to avoid timezone issues
				if result.UnixNano() != tt.expected.UnixNano() {
					t.Errorf("UnmarshalJSON() = %v (%d), want %v (%d)",
						result.Time, result.UnixNano(),
						tt.expected.Time, tt.expected.UnixNano())
				}
			}
		})
	}
}

func TestUnixNanoTime_String(t *testing.T) {
	tests := []struct {
		name     string
		input    UnixNanoTime
		expected string
	}{
		{
			name:     "epoch time",
			input:    UnixNanoTime{Time: time.Unix(0, 0)},
			expected: "0",
		},
		{
			name:     "specific time",
			input:    UnixNanoTime{Time: time.Date(2023, 1, 1, 12, 0, 0, 123456789, time.UTC)},
			expected: `"1672574400123456789"`,
		},
		{
			name:     "small time",
			input:    UnixNanoTime{Time: time.Unix(0, 1000000000)},
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

func TestUnixNanoTime_RoundTrip(t *testing.T) {
	times := []time.Time{
		time.Unix(0, 0),                  // epoch
		time.Unix(1, 0),                  // 1 second after epoch
		time.Unix(0, 1000000000),         // 1 second in nanoseconds
		time.Unix(1672574400, 123456789), // 2023-01-01 12:00:00.123456789 UTC
		time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC), // end of 2023
		time.Unix(-1, 0),                  // before epoch
		time.Unix(0, 9223372036854775807), // MaxInt64 nanoseconds
	}

	for _, original := range times {
		t.Run("round trip", func(t *testing.T) {
			ut := UnixNanoTime{Time: original}
			data, err := ut.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			var unmarshaled UnixNanoTime
			err = unmarshaled.UnmarshalJSON(data)
			if err != nil {
				t.Errorf("UnmarshalJSON() error = %v", err)
				return
			}

			if unmarshaled.UnixNano() != original.UnixNano() {
				t.Errorf("Round trip failed: original = %v (%d), unmarshaled = %v (%d)",
					original, original.UnixNano(),
					unmarshaled.Time, unmarshaled.UnixNano())
			}
		})
	}
}

func TestUnixNanoTime_Precision(t *testing.T) {
	// Test that nanosecond precision is preserved
	nanoTime := time.Unix(1672574400, 123456789) // specific nanoseconds
	ut := UnixNanoTime{Time: nanoTime}

	data, err := ut.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var unmarshaled UnixNanoTime
	err = unmarshaled.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if unmarshaled.Nanosecond() != nanoTime.Nanosecond() {
		t.Errorf("Nanosecond precision lost: original = %d ns, unmarshaled = %d ns",
			nanoTime.Nanosecond(), unmarshaled.Nanosecond())
	}
}
