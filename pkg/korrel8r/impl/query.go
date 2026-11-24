// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"fmt"
	"net/url"
	"strings"

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

// SplitURLQueryData separates a trailing URL query section from query data.
// It searches question marks from right to left and returns the first non-empty
// suffix accepted by url.ParseQuery. If no valid suffix is found, it returns the
// original data and nil values; malformed candidate sections are reported when no
// valid trailing candidate exists. Callers must resolve language-specific ambiguity
// first (for example, parse an expression before calling this helper).
func SplitURLQueryData(data string) (string, url.Values, error) {
	var parseErr error
	for i := strings.LastIndexByte(data, '?'); i >= 0; i = strings.LastIndexByte(data[:i], '?') {
		section := data[i+1:]
		if section == "" {
			continue
		}
		params, err := url.ParseQuery(section)
		if err != nil {
			if parseErr == nil {
				parseErr = err
			}
			continue
		}
		return data[:i], params, nil
	}
	if parseErr != nil {
		return data, nil, parseErr
	}
	return data, nil, nil
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
