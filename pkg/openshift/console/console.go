// package console helps convert queries to and from console  URLs.
//
// The Query type for domains that support console URLs must implement the Query interface
package console

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

func New(baseURL *url.URL, e *engine.Engine) *Console {
	return &Console{BaseURL: baseURL, e: e}
}

func (c *Console) converter(domain string) Converter {
	for _, a := range []any{c.e.Domain(domain), c.e.Store(domain)} {
		if qc, ok := a.(Converter); ok {
			return qc
		}
	}
	return nil
}

func (c *Console) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	for _, x := range []struct{ prefix, domain string }{
		{"/k8s", "k8s"},
		{"/monitoring/alerts", "alert"},
		{"/monitoring/logs", "logs"},
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

func (c *Console) QueryToConsoleURL(q korrel8r.Query) (*url.URL, error) {
	if qc := c.converter(q.Class().Domain().String()); qc != nil {
		u, err := qc.QueryToConsoleURL(q)
		if err != nil {
			return nil, err
		}
		return c.BaseURL.ResolveReference(u), nil
	}
	return nil, fmt.Errorf("cannot convert query to console URL: q")
}
