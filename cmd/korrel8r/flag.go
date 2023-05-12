// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TimeFlag for flags
type TimeFlag struct{ Time *time.Time }

func (f *TimeFlag) String() string {
	if f.Time != nil {
		return f.Time.String()
	}
	return ""
}

func (f *TimeFlag) Set(s string) (err error) {
	*f.Time, err = time.ParseInLocation(time.RFC3339, s, time.Local)
	return err
}

func (TimeFlag) Type() string { return "timestamp" }

type URLFlag struct{ *url.URL }

func (f *URLFlag) String() string {
	if f.URL != nil {
		return f.URL.String()
	}
	return ""
}

func (f *URLFlag) Set(s string) error {
	var err error
	f.URL, err = url.Parse(s)
	return err
}

func (URLFlag) Type() string { return "URL" }

type EnumFlag struct {
	Value *string
	Enum  []string
}

func NewEnumFlag(enum ...string) *EnumFlag {
	dflt := enum[0]
	return &EnumFlag{Value: &dflt, Enum: enum}
}

func (f *EnumFlag) String() string {
	if f.Value != nil {
		return *f.Value
	}
	return f.Enum[0]
}

func (f *EnumFlag) Set(s string) error {
	for _, e := range f.Enum {
		if e == s {
			*f.Value = e
			return nil
		}
	}
	return fmt.Errorf("invalid value %v, expected %v", s, f.Type())
}

func (f EnumFlag) Type() string { return strings.Join(f.Enum, "|") }
