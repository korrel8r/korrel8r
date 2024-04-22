// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package openshift

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Console convert console URL to/from korrel8r Query.
type Console struct {
	BaseURL *url.URL
	c       client.Client
}

func NewConsole(baseURL *url.URL, c client.Client) *Console {
	return &Console{BaseURL: baseURL, c: c}
}

func (c *Console) URLFromQuery(q korrel8r.Query) (u *url.URL, err error) {
	defer func() {
		u = c.BaseURL.ResolveReference(u)
	}()
	switch q.Class().Domain().Name() {
	case "k8s":
		return c.k8sURL(q)
	case "netflow":
		return c.netflowURL(q)
	case "metric":
		return c.metricURL(q)
	case "alert":
		return c.alertURL(q)
	case "log":
		return c.logURL(q)
	default:
		return nil, fmt.Errorf("cannot convert query to console URL: %v", q)
	}
}

func (c *Console) QueryFromURL(u *url.URL) (q korrel8r.Query, err error) {
	for _, x := range []struct {
		prefix  string
		convert func(*url.URL) (korrel8r.Query, error)
	}{
		{"/k8s", c.k8sQuery},
		{"/search", c.k8sQuery},
		{"/monitoring/alerts", c.alertQuery},
		{"/monitoring/logs", c.logQuery},
		{"/monitoring/query-browser", c.metricQuery},
		{"/netflow-traffic", c.netflowQuery},
	} {
		if strings.HasPrefix(path.Join("/", u.Path), x.prefix) {
			return x.convert(u)
		}
	}
	return nil, fmt.Errorf("Cannot convert console URL to query: %v", u)
}
