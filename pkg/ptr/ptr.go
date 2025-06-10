// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package ptr provides pointer-related functions
package ptr

// To returns a pointer to the value of v for any type.
func To[T any](v T) *T { return &v }

// ToSlice returns a pointer to the slice, or nil if the slice is empty.
func ToSlice[T any](v []T) *[]T {
	if len(v) == 0 {
		return nil
	}
	return &v
}

// ToBool returns a pointer to the value if true, nil if false.
func ToBool(v bool) *bool {
	if v {
		return &v
	}
	return nil
}

// Deref returns the value pointed at, or a zero value if the pointer is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var v T
		return v
	}
	return *p
}
