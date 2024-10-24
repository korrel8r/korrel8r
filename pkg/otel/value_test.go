// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otlp "go.opentelemetry.io/proto/otlp/common/v1"
)

var (
	stringValue = &otlp.AnyValue{Value: &otlp.AnyValue_StringValue{StringValue: "a"}}
	boolValue   = &otlp.AnyValue{Value: &otlp.AnyValue_BoolValue{BoolValue: true}}
	intValue    = &otlp.AnyValue{Value: &otlp.AnyValue_IntValue{IntValue: 9}}
	doubleValue = &otlp.AnyValue{Value: &otlp.AnyValue_DoubleValue{DoubleValue: 9.9}}
	bytesValue  = &otlp.AnyValue{Value: &otlp.AnyValue_BytesValue{BytesValue: []byte{3}}}
	arrayValue  = &otlp.AnyValue{Value: &otlp.AnyValue_ArrayValue{ArrayValue: &otlp.ArrayValue{
		Values: []*otlp.AnyValue{stringValue, intValue},
	}}}
	arrayValue2 = &otlp.AnyValue{Value: &otlp.AnyValue_ArrayValue{ArrayValue: &otlp.ArrayValue{
		Values: []*otlp.AnyValue{arrayValue},
	}}}
	kvlistValue = &otlp.AnyValue{Value: &otlp.AnyValue_KvlistValue{KvlistValue: &otlp.KeyValueList{
		Values: []*otlp.KeyValue{
			{Key: "s", Value: stringValue},
			{Key: "i", Value: intValue},
		}}}}
	kvlistValue2 = &otlp.AnyValue{Value: &otlp.AnyValue_KvlistValue{KvlistValue: &otlp.KeyValueList{
		Values: []*otlp.KeyValue{
			{Key: "kv", Value: kvlistValue},
		}}}}

	testData = []struct {
		name       string
		otlpValue  *otlp.AnyValue
		goValue    any
		marshalled string
	}{
		{"stringValue", stringValue, "a", `{"stringValue":"a"}`},
		{"boolValue", boolValue, true, `{"boolValue":true}`},
		{"intValue", intValue, int64(9), `{"intValue":"9"}`},
		{"doubleValue", doubleValue, 9.9, `{"doubleValue":9.9}`},
		{"bytesValue", bytesValue, []byte{3}, `{"bytesValue":"Aw=="}`},
		{"arrayValue", arrayValue, []any{"a", int64(9)}, `{"arrayValue":{"values":[{"stringValue":"a"},{"intValue":"9"}]}}`},
		{"arrayValue2", arrayValue2, []any{[]any{"a", int64(9)}}, `{"arrayValue":{"values":[{"arrayValue":{"values":[{"stringValue":"a"},{"intValue":"9"}]}}]}}`},
		{"kvlistValue", kvlistValue, KeyValueList{{"s", Value{"a"}}, {"i", Value{int64(9)}}},
			`{"kvlistValue":{"values":[{"key":"s","value":{"stringValue":"a"}},{"key":"i","value":{"intValue":9}}]}}`},
		{"kvlistValue2", kvlistValue2, KeyValueList{{"kv", Value{KeyValueList{{"s", Value{"a"}}, {"i", Value{int64(9)}}}}}},
			`{"kvlistValue":{"values":[{"key":"kv","value":{"kvlistValue":{"values":[{"key":"s","value":{"stringValue":"a"}},{"key":"i","value":{"intValue":9}}]}}}]}}`},
	}
)

func Test_valueOf(t *testing.T) {
	for _, x := range testData {
		t.Run(x.name, func(t *testing.T) { assert.Equal(t, x.goValue, ValueOf(x.otlpValue)) })
	}
}

func TestValue_Unmarshal(t *testing.T) {
	for _, x := range testData {
		t.Run(x.name, func(t *testing.T) {
			var v Value
			err := json.Unmarshal([]byte(x.marshalled), &v)
			if assert.NoError(t, err, "%v", x.marshalled) {
				assert.Equal(t, x.goValue, v.Value)
			}
		})
	}
}

func kv(k string, v any) KeyValue { return KeyValue{Key: k, Value: Value{v}} }

func TestAttributes_UnmarshalJSON(t *testing.T) {
	var a KeyValueList
	const data = `[
{"key":"b", "value":{"bytesValue":"Aw=="}},
{"key":"d", "value":{"doubleValue":9.9}},
{"key":"a", "value":{"arrayValue":{"values":[{"stringValue":"a"},{"intValue":"9"}]}}}
]`
	err := json.Unmarshal([]byte(data), &a)
	require.NoError(t, err)
	assert.Equal(t, a, KeyValueList{kv("b", []byte{3}), kv("d", 9.9), kv("a", []any{"a", int64(9)})})
}

func TestAttributes_Map(t *testing.T) {
	l := KeyValueList{kv("s", "a"), kv("a", []any{"a", int64(9)})}
	want := map[string]any{"a": []any{"a", int64(9)}, "s": "a"}
	assert.Equal(t, want, l.Map())
}
