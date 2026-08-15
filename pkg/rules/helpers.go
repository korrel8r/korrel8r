// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Empty returns true if v is nil, false, zero, or a zero-length string, slice, or map.
func Empty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	default:
		return rv.IsZero()
	}
}

// Fail panics with a formatted error.
func Fail(format string, args ...any) {
	panic(fmt.Errorf(format, args...))
}

// FailErr panics if err is not nil.
func FailErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Require returns v if it is not [Empty], calls [Fail] otherwise.
func Require[T any](v T) T {
	if Empty(v) {
		Fail("missing required value")
	}
	return v
}

// RequireAll calls [Fail] if any values are [Empty].
func RequireAll(values ...any) {
	for _, v := range values {
		if Empty(v) {
			Fail("missing required values")
		}
	}
}

// ToJSON serializes v to a JSON string, panics on error.
func ToJSON(v any) string {
	b, err := json.Marshal(v)
	FailErr(err)
	return string(b)
}

// Default returns dflt if [Empty](v), else returns v.
func Default[T any](dflt, v T) T {
	if Empty(v) {
		return dflt
	}
	return v
}
