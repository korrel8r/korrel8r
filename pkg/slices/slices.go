// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package slices

import (
	"fmt"
)

func Transform[T, U any](t []T, f func(T) U) []U {
	u := make([]U, len(t))
	for i := range t {
		u[i] = f(t[i])
	}
	return u
}

func Strings[T any](t []T) []string {
	return Transform(t, func(v T) string { return fmt.Sprintf("%v", v) })
}
