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
// # Query
//
// Query data is a PromQL [instant vector selector], for example:
//
//	metric:metric:http_requests_total{environment=~"staging|testing",method!="GET"}
//
// # Store
//
// Prometheus is the store, store configuration:
//
//	domain: metric
//	metric: URL_OF_PROMETHEUS
//
// [Metric]: https://pkg.go.dev/github.com/prometheus/common@v0.45.0/model#Metric
// [instant vector selector]: https://prometheus.io/docs/prometheus/latest/querying/basics/#instant-vector-selectors
package metric

// TODO: doc comment needs to show model.Metric structure or link to it properly.
// FIXME: metrics are only usable as goals.

import (
	"context"
	"net/http"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/prometheus/client_golang/api"
	prometheus "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	Domain = domain{}
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query("")
	_ korrel8r.Store  = &Store{}
)

type domain struct{}

func (domain) Name() string                     { return "metric" }
func (d domain) String() string                 { return d.Name() }
func (domain) Description() string              { return "Time-series of measured values" }
func (domain) Class(name string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class        { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	_, qs, err := impl.ParseQueryString(d, s)
	return Query(qs), err
}

const StoreKeyMetricURL = "metric"

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

func (c Class) Domain() korrel8r.Domain          { return Domain }
func (c Class) Name() string                     { return Domain.Name() }
func (c Class) String() string                   { return impl.ClassString(c) }
func (c Class) Description() string              { return "A set of label:value pairs identifying a time-series." }
func (c Class) New() korrel8r.Object             { var obj Object; return obj }
func (c Class) Preview(o korrel8r.Object) string { return Preview(o) }

type Object = model.Metric

func Preview(o korrel8r.Object) string {
	return impl.Preview(o, func(o Object) string { return o.String() })
}

// Query is a PromQL instance vector query.
type Query string

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return impl.QueryString(q) }

type Store struct{ api prometheus.API }

func NewStore(baseURL string, hc *http.Client) (korrel8r.Store, error) {
	c, err := api.NewClient(api.Config{Address: baseURL, Client: hc})
	if err != nil {
		return nil, err
	}
	return &Store{prometheus.NewAPI(c)}, nil
}

func (s *Store) Domain() korrel8r.Domain { return Domain }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	if _, err := impl.TypeAssert[Query](query); err != nil {
		return err
	}
	labelSets, _, err := s.api.Series(ctx, []string{query.Data()}, ptr.ValueOf(c.Start), ptr.ValueOf(c.End))
	if err != nil {
		return err
	}
	for i, v := range labelSets {
		// FIXME Next release of "github.com/prometheus/client_golang/api/prometheus/v1" will include
		// WithLimit(*c.Limit) to set limit in the query. Until then, ignore excess results.
		if c.Limit != nil && uint(i) >= *c.Limit {
			break
		}
		result.Append(v)
	}
	return nil
}
