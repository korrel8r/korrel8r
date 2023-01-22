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
	for _, x := range [][2]string{
		{"/k8s", "k8s"},
		{"/monitoring/alerts", "alert"},
		{"/monitoring/logs", "loki"},
		{"/monitoring/query-browser", "metric"},
	} {
		if strings.HasPrefix(path.Join("/", u.Path), x[0]) {
			if qc := c.converter(x[1]); qc != nil {
				return qc.ConsoleURLToQuery(u)
			}
		}
	}
	return nil, fmt.Errorf("cannot convert console URL to query: %v", u)
}

func (c *Console) QueryToConsoleURL(q korrel8r.Query) (*url.URL, error) {
	if qc := c.converter(q.Class().Domain().String()); qc != nil {
		ref, err := qc.QueryToConsoleURL(q)
		return c.BaseURL.ResolveReference(ref), err
	}
	return nil, fmt.Errorf("cannot convert query to console URL: q")
}
