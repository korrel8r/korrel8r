// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package types

import (
	"encoding/json"
	"math"
	"strconv"
)

// ProtobufInt is an int64 that (un)marshlas using JSON protobuf encoding
// which allows the JSON representation to be a number or a string.
type ProtobufInt int64

// UnmarshalJSON using protobuf rules, allows JSON number or string.
func (i *ProtobufInt) UnmarshalJSON(data []byte) error {
	// Remove quotes if present
	if len(data) > 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	n, err := strconv.ParseInt(string(data), 10, 64)
	*i = ProtobufInt(n)
	return err
}

// MarshalJSON using protobuf rules, as number if < 32 bits as string if greater.
func (i ProtobufInt) MarshalJSON() ([]byte, error) {
	if i < math.MaxInt32 && i > math.MinInt32 {
		return json.Marshal(int64(i))
	}
	// Use JSON string for > 32 bits as per protobuf rules.
	return json.Marshal(strconv.FormatInt(int64(i), 10))
}
