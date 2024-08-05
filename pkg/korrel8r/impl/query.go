// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// ParseQueryString parses a query string into class and data parts.
func ParseQueryString(domain korrel8r.Domain, query string) (class korrel8r.Class, data string, err error) {
	d, c, q := QuerySplit(query)
	if q == "" {
		return nil, "", fmt.Errorf("invalid query: %v", query)
	}
	if d != domain.Name() {
		return nil, "", fmt.Errorf("wrong query domain, want %v: %v", domain, query)
	}
	class = domain.Class(c)
	if class == nil {
		return nil, "", korrel8r.ClassNotFoundError{Domain: domain, Class: c}
	}
	return class, q, nil
}

// UnmarshalQueryString unmarshals JSON query string to Go values.
// T is the type to use to unmarshal the query data part.
func UnmarshalQueryString[T any](domain korrel8r.Domain, query string) (korrel8r.Class, T, error) {
	c, qs, err := ParseQueryString(domain, query)
	var data T
	if err != nil {
		return nil, data, err
	}
	data, err = UnmarshalAs[T]([]byte(qs))
	if err != nil {
		return c, data, fmt.Errorf("invalid query: %w: %v", err, qs)
	}
	return c, data, nil
}
