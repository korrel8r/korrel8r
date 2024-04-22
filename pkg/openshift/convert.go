// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package openshift

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

func (c *Console) netflowURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[netflow.Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Add("q", q.Data()+"|json")
	v.Add("tenant", "network")
	return &url.URL{Path: "/netflow-traffic", RawQuery: v.Encode()}, nil
}

func (c *Console) netflowQuery(u *url.URL) (korrel8r.Query, error) {
	q := u.Query().Get("q")
	return netflow.NewQuery(q), nil
}

func (c *Console) logURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[log.Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{}
	v.Add("q", q.Data()+"|json")
	v.Add("tenant", q.Class().Name())
	return &url.URL{Path: "/monitoring/logs", RawQuery: v.Encode()}, nil
}

func (c *Console) logQuery(u *url.URL) (korrel8r.Query, error) {
	logQL := u.Query().Get("q")
	if class := log.Domain.Class(u.Query().Get("tenant")); class != nil {
		return log.NewQuery(class.(log.Class), logQL), nil
	}
	return log.NewQuery("", logQL), nil
}

func (c *Console) metricQuery(u *url.URL) (korrel8r.Query, error) {
	promQL := u.Query().Get("query0")
	if promQL == "" || !strings.HasPrefix(u.Path, "/monitoring/query-browser") {
		return nil, fmt.Errorf("not a metric console query: %v", u)
	}
	return metric.Query{PromQL: promQL}, nil
}

func (c *Console) metricURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[metric.Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{"query0": []string{q.PromQL}}
	return &url.URL{Path: "/monitoring/query-browser", RawQuery: v.Encode()}, nil
}

func (c *Console) alertQuery(u *url.URL) (korrel8r.Query, error) {
	m := map[string]string{}
	uq := u.Query()
	for k := range uq {
		m[k] = uq.Get(k)
	}
	return alert.Query(m), nil
}

func (c *Console) alertURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[alert.Query](query)
	if err != nil {
		return nil, err
	}
	uq := url.Values{
		"rowFilter-alert-state": []string{""}, // do not filter by alert state.
	}
	alertFilter := make([]string, 0, len(q))
	for k, v := range q {
		alertFilter = append(alertFilter, fmt.Sprintf("%s=%s", k, v))
	}
	uq.Add("alerts", strings.Join(alertFilter, ","))

	return &url.URL{
		Path:     "/monitoring/alerts",
		RawQuery: uq.Encode(),
	}, nil
}
