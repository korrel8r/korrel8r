// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"slices"
	"strings"
)

// EnumFlagValue is a flag value that validates its value is one of a set of allowed values.
type EnumFlagValue struct {
	Allowed []string
	Value   string
}

// EnumFlag returns an enum flag value with the allowed strings. First one is the default.
func EnumFlag(allowed ...string) *EnumFlagValue {
	return &EnumFlagValue{Allowed: allowed, Value: allowed[0]}
}
func (f *EnumFlagValue) String() string { return f.Value }
func (f *EnumFlagValue) Type() string   { return fmt.Sprintf("enum(%v)", strings.Join(f.Allowed, ",")) }
func (f *EnumFlagValue) Set(s string) error {
	if !slices.Contains(f.Allowed, s) {
		return fmt.Errorf("invalid value %q: must be one of %v", s, f.Allowed)
	}
	f.Value = s
	return nil
}
