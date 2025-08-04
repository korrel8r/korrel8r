// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package types

import (
	"encoding/json"
	"time"
)

// MilliDuration wraps time.Duration to (un)marshal as milliseconds.
type MilliDuration struct{ time.Duration }

// MarshalJSON as milliseconds.
func (d MilliDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(ProtobufInt(d.Milliseconds()))
}

// UnarshalJSON as milliseconds.
func (d *MilliDuration) UnmarshalJSON(data []byte) error {
	var n ProtobufInt
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
	return json.Marshal(ProtobufInt(d.Nanoseconds()))
}

// UnmarshalJSON as nanoseconds
func (d *NanoDuration) UnmarshalJSON(data []byte) error {
	var n ProtobufInt
	err := json.Unmarshal(data, &n)
	d.Duration = time.Duration(n)
	return err
}

// marshalSting shorthand template for string value of json.Marshal(v)
func marshalSting[T json.Marshaler](v T) string {
	b, _ := v.MarshalJSON()
	return string(b)
}
