// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package enumflag is custom flag value that allows one of a list of strings.
// Implements standard flag.Value and cobra pflag.Value
package enumflag

import (
	"fmt"
	"slices"
	"strings"
)

type Value struct {
	Value   string
	Allowed []string
}

func (v *Value) String() string { return v.Value }

func (v *Value) Set(x string) error {
	if slices.Index(v.Allowed, x) < 0 {
		return fmt.Errorf("expected one of: %v", v.Allowed)
	}
	v.Value = x
	return nil
}

func (v *Value) DocString(msg string) string {
	w := &strings.Builder{}
	if msg != "" {
		fmt.Fprintf(w, "%v: ", msg)
	}
	fmt.Fprintf(w, "One of %v", v.Allowed)
	return w.String()
}

func (v *Value) Type() string { return "string" }

func New(value string, allowed []string) *Value {
	slices.Sort(allowed)
	return &Value{Allowed: allowed, Value: value}
}
