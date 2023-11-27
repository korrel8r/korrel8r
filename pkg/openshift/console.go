// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package console helps convert queries to and from console  URLs.
//
// The Query type for domains that support console URLs must implement the Query interface
package openshift

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Converter interface must be implemented by korrel8r.Domain and/or korrel8r.Store implementations
// that support console URL conversion.
type Converter interface {
	QueryToConsoleURL(korrel8r.Query) (*url.URL, error)
	ConsoleURLToQuery(*url.URL) (korrel8r.Query, error)
}

type Console struct {
	BaseURL *url.URL
	e       *engine.Engine
}

func NewConsole(baseURL *url.URL, e *engine.Engine) *Console {
	return &Console{BaseURL: baseURL, e: e}
}

func (c *Console) converter(domain string) Converter {
	if d := c.e.Domain(domain); d != nil {
		if converter, ok := d.(Converter); ok {
			return converter
		}
		for _, s := range c.e.StoresFor(d) {
			if converter, ok := s.(Converter); ok {
				return converter
			}
		}
	}
	return nil
}

func (c *Console) ConsoleURLToQuery(u *url.URL) (q korrel8r.Query, err error) {
	for _, x := range []struct{ prefix, domain string }{
		{"/k8s", "k8s"},
		{"/search", "k8s"},
		{"/monitoring/alerts", "alert"},
		{"/monitoring/logs", "log"},
		{"/monitoring/query-browser", "metric"},
	} {
		if strings.HasPrefix(path.Join("/", u.Path), x.prefix) {
			if qc := c.converter(x.domain); qc != nil {
				return qc.ConsoleURLToQuery(u)
			}
			break
		}
	}
	return nil, fmt.Errorf("cannot convert console URL to query: %v", u)
}

func (c *Console) QueryToConsoleURL(q korrel8r.Query) (u *url.URL, err error) {
	if qc := c.converter(q.Class().Domain().Name()); qc != nil {
		u, err := qc.QueryToConsoleURL(q)
		if err != nil {
			return nil, err
		}
		return c.BaseURL.ResolveReference(u), nil
	}
	return nil, fmt.Errorf("cannot convert query to console URL: q")
}
