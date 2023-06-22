// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package impl provides helper types and functions for implementing a korrel8r domain.
package impl

import (
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

func Query(s string, q korrel8r.Query) (korrel8r.Query, error) {
	err := yaml.Unmarshal([]byte(s), q)
	if q.Class() == nil {
		return nil, fmt.Errorf("query has no class: %+v", q)
	}
	return q, err
}
