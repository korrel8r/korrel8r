// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otel

// KeyValueList holds attributes as a list of key:value pairs.
type KeyValueList []KeyValue

// KeyValue is an attribute key:value pair.
type KeyValue struct {
	Key   string `json:"key"`
	Value Value  `json:"value"`
}

// Map converts a KeyValueList returns into a new map[string]any
func (l KeyValueList) Map() map[string]any {
	m := map[string]any{}
	for _, kv := range l {
		m[kv.Key] = kv.Value.Value
	}
	return m
}
