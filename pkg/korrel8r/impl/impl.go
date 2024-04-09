// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package impl provides helper types and functions for implementing a korrel8r domain.
package impl

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"sigs.k8s.io/yaml"
)

// TypeName returns the name of the static type of its argument, which may be an interface.
func TypeName[T any](v T) string { return reflect.TypeOf((*T)(nil)).Elem().String() }

// TypeAssert does a type assertion and returns a useful error if it fails.
func TypeAssert[T any](x any) (v T, err error) {
	v, ok := x.(T)
	if !ok {
		err = fmt.Errorf("wrong type: want %v, got (%T)(%#v)", TypeName(v), x, x)
	}
	return v, err
}

// ParseQueryString parses a query string into class and query parts.
func ParseQueryString(domain korrel8r.Domain, query string) (class korrel8r.Class, data string, err error) {
	d, c, q := QuerySplit(query)
	if q == "" {
		return nil, "", fmt.Errorf("invalid query: %v", query)
	}
	if d != domain.Name() {
		return nil, "", fmt.Errorf("wrong query domain, want %v: %v", domain, query)
	}
	class = domain.Class(c)
	if class == nil {
		return nil, "", korrel8r.ClassNotFoundErr{Domain: domain, Class: c}
	}
	return class, q, nil
}

func UnmarshalQueryString(domain korrel8r.Domain, query string, data any) (korrel8r.Class, error) {
	c, qs, err := ParseQueryString(domain, query)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal([]byte(qs), data); err != nil {
		return c, fmt.Errorf("invalid query: %w: %v", err, qs)
	}
	return c, nil
}

// JSONString returns the JSON marshaled string from v, or the error message if marshal fails
func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", err.Error())
	}
	return string(b)
}
