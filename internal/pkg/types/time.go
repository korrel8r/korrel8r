// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package types

import (
	"encoding/json"
	"time"
)

// UnixNanoTime wraps time.Time to (un)marshal as nanosecond Unix time
// using Protobuf Int encoding.
type UnixNanoTime struct{ time.Time }

// String matches json.Marshal.
func (t UnixNanoTime) String() string { return marshalSting(t) }

// MarshalJSON as a JSON string of nanosecond Unix time.
func (t UnixNanoTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ProtobufInt(t.UnixNano()))
}

// UnmarshalJSON from a JSON string or number with nanosecond Unix time.
func (t *UnixNanoTime) UnmarshalJSON(b []byte) error {
	var n ProtobufInt
	err := json.Unmarshal(b, &n)
	t.Time = time.Unix(0, int64(n))
	return err
}
