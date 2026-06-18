// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package impl provides helper types and functions for implementing a korrel8r domain.
package impl

import (
	"fmt"
	"reflect"

	"github.com/korrel8r/korrel8r/internal/pkg/yaml"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// TypeName returns the name of the static type of its argument, which may be an interface.
func TypeName[T any](v T) string { return reflect.TypeFor[T]().String() }

// TypeAssert does a type assertion and returns a useful error if it fails.
func TypeAssert[T any](x any) (v T, err error) {
	v, ok := x.(T)
	if !ok {
		err = fmt.Errorf("wrong type: want %v, got (%T)(%#v)", TypeName(v), x, x)
	}
	return v, err
}

// UnmarshalAs can unmarshal YAML or JSON, returns the new value.
func UnmarshalAs[T any](b []byte) (T, error) {
	var v T
	err := Unmarshal(b, &v)
	return v, err
}

// Unmarshal can unmarshal YAML or JSON.
func Unmarshal(b []byte, v any) error { return yaml.UnmarshalStrict(b, &v) }

// AssertDomainTypes is a compile-time assertion that types meet the korrel8r interface requirements.
func AssertDomainTypes[
	Class interface {
		korrel8r.Class
		comparable
	},
	Query interface {
		korrel8r.Query
		comparable
	}](korrel8r.Domain, korrel8r.Object, Class, Query, korrel8r.Store) any {
	return nil
}

func AssertQuery[Query interface {
	korrel8r.Query
	comparable
}](Query) any {
	return nil
}

func AssertClass[Class interface {
	korrel8r.Class
	comparable
}](Class) any {
	return nil
}
