// package must contains functions to handle errors via panic
package must

import (
	"fmt"
	"runtime"
)

// ErrorIf returns nil if err == nil, otherwise returns fmt.Errof(format, args)
func ErrorIf(err error, format string, args ...any) error {
	if err != nil {
		return fmt.Errorf(format, args...)
	}
	return nil
}

// Must panics if err != nil.
// If format is provided, panic contains fmt.Errorf(format...) else it contains err.
func Must(err error, format ...any) {
	if len(format) > 0 {
		err = ErrorIf(err, format[0].(string), format[1:]...)
	}
	if err != nil {
		panic(err)
	}
}

// Must1 calls Must(err), then returns v.
func Must1[T any](v T, err error) T { Must(err); return v }

// Must2 calls Must(err), then returns (v1, v2).
func Must2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) { Must(err); return v1, v2 }

// NormalError if r is a non-runtime error, return the error, else return nil.
func NormalError(r any) error {
	switch err := r.(type) {
	case runtime.Error: // Don't catch runtime errors.
		return nil
	case error:
		return err
	default:
		return nil
	}
}
