package impl

import (
	"fmt"
	"reflect"
)

// Domain is a base type for korrel8r.Domain implementations
type Domain string

func (d Domain) String() string { return string(d) }

// TypeName returns the name of the static type of its argument, which may be an interface.
func TypeName[T any](v T) string { return reflect.TypeOf((*T)(nil)).Elem().String() }

// TypeAssert does a type assertion and returns a useful error if it fails.
func TypeAssert[T any](x any) (v T, err error) {
	v, ok := x.(T)
	if !ok {
		err = fmt.Errorf("wrong type: want %v, got %T = %#v", TypeName(v), x, x)
	}
	return v, err
}
