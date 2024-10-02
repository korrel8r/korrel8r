// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otel

import (
	otlpcommon "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// Value is an any that can convert to/from and (un)marshal as an OTEL [otlpcommon.AnyValue]
type Value struct{ Value any }

// UnmarshalJSON unmarshal from the OTLP JSON protobuf value format.
func (v *Value) UnmarshalJSON(data []byte) error {
	var av otlpcommon.AnyValue
	err := protojson.Unmarshal(data, &av)
	v.Value = ValueOf(&av)
	return err
}

// MarshalJSON panics, not implemented.
func (v Value) MarshalJSON() ([]byte, error) { panic("not implemented") }

// TODO MarshalJSON not needed now but should be implemented for sanity.

// ValueOf converts an [otlp.AnyValue] to a [Value].
func ValueOf(v *otlpcommon.AnyValue) any {
	switch v := v.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		return v.StringValue
	case *otlpcommon.AnyValue_BoolValue:
		return v.BoolValue
	case *otlpcommon.AnyValue_IntValue:
		return v.IntValue
	case *otlpcommon.AnyValue_DoubleValue:
		return v.DoubleValue
	case *otlpcommon.AnyValue_ArrayValue:
		a := make([]any, len(v.ArrayValue.Values))
		for i, v := range v.ArrayValue.Values {
			a[i] = ValueOf(v)
		}
		return a
	case *otlpcommon.AnyValue_KvlistValue:
		var a KeyValueList
		for _, kv := range v.KvlistValue.Values {
			a = append(a, KeyValue{Key: kv.Key, Value: Value{ValueOf(kv.Value)}})
		}
		return a
	case *otlpcommon.AnyValue_BytesValue:
		return v.BytesValue
	}
	return nil
}
