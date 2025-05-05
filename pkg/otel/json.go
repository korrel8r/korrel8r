// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otel

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// Int is an int64 that (un)marshlas using JSON protobuf encoding
type Int int64

// UnmarshalJSON using protobuf rules, allows JSON number or string.
func (i *Int) UnmarshalJSON(data []byte) error {
	// Remove quotes if present
	if len(data) > 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	n, err := strconv.ParseInt(string(data), 10, 64)
	*i = Int(n)
	return err
}

// MarshalJSON using protobuf rules, as number if < 32 bits as string if greater.
func (i Int) MarshalJSON() ([]byte, error) {
	if i < math.MaxInt32 && i > math.MinInt32 {
		return json.Marshal(int64(i))
	}
	// Use JSON string for > 32 bits as per protobuf rules.
	return json.Marshal(strconv.FormatInt(int64(i), 10))
}

// UnixNanoTime wraps time.Time to (un)marshal as nanosecond Unix time.
type UnixNanoTime struct{ time.Time }

// String matches json.Marshal.
func (t UnixNanoTime) String() string { return marshalSting(t) }

// MarshalJSON as a JSON string of nanosecond Unix time.
func (t UnixNanoTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(Int(t.UnixNano()))
}

// UnmarshalJSON from a JSON string or number with nanosecond Unix time.
func (t *UnixNanoTime) UnmarshalJSON(b []byte) error {
	var n Int
	err := json.Unmarshal(b, &n)
	t.Time = time.Unix(0, int64(n))
	return err
}

// MilliDuration wraps time.Duration to (un)marshal as milliseconds.
type MilliDuration struct{ time.Duration }

// MarshalJSON as milliseconds.
func (d MilliDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(Int(d.Milliseconds()))
}

// UnarshalJSON as milliseconds.
func (d *MilliDuration) UnmarshalJSON(data []byte) error {
	var n Int
	err := json.Unmarshal(data, &n)
	d.Duration = time.Duration(int64(n) * int64(time.Millisecond))
	return err
}

// String matches json.Marshal.
func (d MilliDuration) String() string { return marshalSting(d) }

// NanoDuration wraps time.Duration to (un)marshal as nanoseconds.
type NanoDuration struct{ time.Duration }

// String matches json.Marshal.
func (d NanoDuration) String() string { return marshalSting(d) }

// MarshalJSON as nanoseconds
func (d NanoDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(Int(d.Nanoseconds()))
}

// UnmarshalJSON as nanoseconds
func (d *NanoDuration) UnmarshalJSON(data []byte) error {
	var n Int
	err := json.Unmarshal(data, &n)
	d.Duration = time.Duration(n)
	return err
}

// marshalSting shorthand template for string value of json.Marshal(v)
func marshalSting[T json.Marshaler](v T) string {
	b, _ := v.MarshalJSON()
	return string(b)
}
