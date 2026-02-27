// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// ParseQuery parses a query string into class and selector.
func ParseQuery(domain korrel8r.Domain, query string) (class korrel8r.Class, selector string, err error) {
	d, c, q, err := korrel8r.QuerySplit(query)
	if err != nil {
		return nil, "", err
	}
	if d != domain.Name() {
		return nil, "", fmt.Errorf("wrong domain, want %v: %v", domain.Name(), query)
	}
	class = domain.Class(c)
	if class == nil {
		return nil, "", korrel8r.NewClassNotFoundError(d, c)
	}
	return class, q, nil
}

// UnmarshalQueryString unmarshals JSON query string to Go values.
// T is the type to use to unmarshal the query data part.
func UnmarshalQueryString[T any](domain korrel8r.Domain, query string) (c korrel8r.Class, data T, err error) {
	c, qs, err := ParseQuery(domain, query)
	if err != nil {
		return nil, data, err
	}
	err = Unmarshal([]byte(qs), &data)
	if err != nil {
		return nil, data, fmt.Errorf("invalid query: %w: %v", err, qs)
	}
	return c, data, nil
}
