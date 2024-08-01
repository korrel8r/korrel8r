// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package ptr provides pointer-related functions
package ptr

// To returns a pointer to the value of v for any type.
func To[T any](v T) *T { return &v }

// ValueOf returns the value pointed at, or a zero value if the pointer is nil.
func ValueOf[T any](p *T) T {
	if p == nil {
		var v T
		return v
	}
	return *p
}
