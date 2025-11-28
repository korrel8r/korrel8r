// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package metric represents Prometheus metric time-series as objects.
//
// # Class
//
// There is only one class: `metric:metric`
//
// # Object
//
// A [Metric] is a time series identified by a label set. Korrel8r does not consider load the sample
// data for a time series, or use it in rules. If a korrel8r search time constraints, then metrics
// that have no values that meet the constraint are ignored.
//
// # Store
//
// Prometheus is the store, store configuration:
//
//	domain: metric
//	metric: URL_OF_PROMETHEUS
//
// [Metric]: https://pkg.go.dev/github.com/prometheus/common@v0.45.0/model#Metric
//
// [instant vector selector]: https://prometheus.io/docs/prometheus/latest/querying/basics/#instant-vector-selectors
package metric

// TODO: doc comment needs to show model.Metric structure or link to it properly.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/common/model"
)

const (
	name        = "metric"
	description = "Time-series of measured values"
)

var (
	Domain = &domain{Domain: impl.NewDomain(name, description, Class{})}

	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query("")
	_ korrel8r.Store  = &Store{}
	_ korrel8r.Object = Object{}
)

type domain struct{ *impl.Domain }

func (d domain) Query(s string) (korrel8r.Query, error) {
	_, qs, err := impl.ParseQuery(d, s)
	return Query(qs), err
}

const StoreKeyMetricURL = name

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	return NewStore(cs[StoreKeyMetricURL], hc)
}

type Class struct{} // Singleton class

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return name }
func (c Class) String() string          { return korrel8r.ClassString(c) }

func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) Preview(o korrel8r.Object) string            { return Preview(o) }
func (c Class) ID(o korrel8r.Object) any {
	if o := o.(Object); o != nil {
		return o.Fingerprint()
	}
	return nil
}

type Object = model.Metric

func Preview(o korrel8r.Object) string {
	return impl.Preview(o, func(o Object) string { return o.String() })
}

type Store struct {
	*http.Client
	baseURL *url.URL
	*impl.Store
}

func NewStore(baseURL string, hc *http.Client) (korrel8r.Store, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Store{Client: hc, baseURL: u.JoinPath("/api/v1"), Store: impl.NewStore(Domain)}, nil
}

func (s *Store) Domain() korrel8r.Domain { return Domain }

type response struct {
	Status string `json:"status"`
	Data   []model.Metric
}

func (s *Store) Get(ctx context.Context, kquery korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	query, err := impl.TypeAssert[Query](kquery)
	if err != nil {
		return err
	}
	// NOTE: Store does not use github.com/prometheus/client_golang because the current version v1.19.1
	// does not allow setting the "limit" query parameter. Hand code the REST query.
	q := url.Values{}
	selectors, err := query.Selectors()
	if err != nil {
		return err
	}
	for _, selector := range selectors {
		q.Add("match[]", selector)
	}
	if c != nil {
		if c.Start != nil && !c.Start.IsZero() {
			q.Set("start", formatTime(*c.Start))
		}
		if c.End != nil && !c.End.IsZero() {
			q.Set("end", formatTime(*c.End))
		}
		if c.Limit != nil && *c.Limit > 0 {
			q.Set("limit", strconv.Itoa(*c.Limit))
		}
	}
	u := s.baseURL.JoinPath("series")
	u.RawQuery = q.Encode()
	var r response
	if err := impl.Get(ctx, u, s.Client, &r); err != nil {
		return err
	}
	if r.Status != "success" {
		return fmt.Errorf("GET %v: unexpected status: %v", u, r.Status)
	}
	for _, m := range r.Data {
		result.Append(m)
	}
	return nil
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
