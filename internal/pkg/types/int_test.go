// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package types

import (
	"math"
	"testing"
)

func TestInt_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    ProtobufInt
		expected string
	}{
		{
			name:     "small positive number",
			input:    ProtobufInt(42),
			expected: "42",
		},
		{
			name:     "small negative number",
			input:    ProtobufInt(-42),
			expected: "-42",
		},
		{
			name:     "zero",
			input:    ProtobufInt(0),
			expected: "0",
		},
		{
			name:     "max 32-bit int",
			input:    ProtobufInt(math.MaxInt32),
			expected: `"2147483647"`,
		},
		{
			name:     "min 32-bit int",
			input:    ProtobufInt(math.MinInt32),
			expected: `"-2147483648"`,
		},
		{
			name:     "large positive number (as string)",
			input:    ProtobufInt(math.MaxInt32 + 1),
			expected: `"2147483648"`,
		},
		{
			name:     "large negative number (as string)",
			input:    ProtobufInt(math.MinInt32 - 1),
			expected: `"-2147483649"`,
		},
		{
			name:     "very large number (as string)",
			input:    ProtobufInt(9223372036854775807), // MaxInt64
			expected: `"9223372036854775807"`,
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

func TestInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ProtobufInt
		wantErr  bool
	}{
		{
			name:     "number format",
			input:    "42",
			expected: ProtobufInt(42),
			wantErr:  false,
		},
		{
			name:     "negative number format",
			input:    "-42",
			expected: ProtobufInt(-42),
			wantErr:  false,
		},
		{
			name:     "string format",
			input:    `"123"`,
			expected: ProtobufInt(123),
			wantErr:  false,
		},
		{
			name:     "negative string format",
			input:    `"-456"`,
			expected: ProtobufInt(-456),
			wantErr:  false,
		},
		{
			name:     "zero",
			input:    "0",
			expected: ProtobufInt(0),
			wantErr:  false,
		},
		{
			name:     "large number as string",
			input:    `"9223372036854775807"`,
			expected: ProtobufInt(9223372036854775807),
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
			var result ProtobufInt
			err := result.UnmarshalJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("UnmarshalJSON() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestInt_RoundTrip(t *testing.T) {
	tests := []ProtobufInt{
		ProtobufInt(0),
		ProtobufInt(42),
		ProtobufInt(-42),
		ProtobufInt(math.MaxInt32),
		ProtobufInt(math.MinInt32),
		ProtobufInt(math.MaxInt32 + 1),
		ProtobufInt(math.MinInt32 - 1),
		ProtobufInt(9223372036854775807),
		ProtobufInt(-9223372036854775808),
	}

	for _, original := range tests {
		t.Run("round trip", func(t *testing.T) {
			data, err := original.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			var unmarshaled ProtobufInt
			err = unmarshaled.UnmarshalJSON(data)
			if err != nil {
				t.Errorf("UnmarshalJSON() error = %v", err)
				return
			}

			if unmarshaled != original {
				t.Errorf("Round trip failed: original = %d, unmarshaled = %d", original, unmarshaled)
			}
		})
	}
}

func TestMarshalSting_Int(t *testing.T) {
	tests := []struct {
		name     string
		input    ProtobufInt
		expected string
	}{
		{
			name:     "small number",
			input:    ProtobufInt(42),
			expected: "42",
		},
		{
			name:     "large number as string",
			input:    ProtobufInt(math.MaxInt32 + 1),
			expected: `"2147483648"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marshalSting(tt.input)
			if result != tt.expected {
				t.Errorf("marshalSting() = %s, want %s", result, tt.expected)
			}
		})
	}
}
