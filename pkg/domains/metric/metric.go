// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package metric is the domain of prometheus metrics.
//
// FIXME metrics are only usable as goals
package metric

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	Domain = domain{}
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = &Query{}
	_ korrel8r.Store  = &Store{}
)

func init() {
	korrel8r.Domains["metric"] = Domain
}

type domain struct{}

func (domain) String() string                   { return "metric" }
func (domain) Class(name string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class        { return []korrel8r.Class{Class{}} }
func (domain) UnmarshalQuery(r []byte) (korrel8r.Query, error) {
	return impl.UnmarshalQuery(r, &Query{})
}

const (
	consolePath = "/monitoring/query-browser"
	promQLParam = "query0"
)

func (domain) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	promQL := u.Query().Get(promQLParam)
	if promQL == "" || !strings.HasPrefix(u.Path, consolePath) {
		return nil, fmt.Errorf("not a metric console query: %v", u)
	}
	return &Query{PromQL: promQL}, nil
}

func (domain) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return nil, err
	}
	v := url.Values{promQLParam: []string{q.PromQL}}
	return &url.URL{Path: consolePath, RawQuery: v.Encode()}, nil
}

type Class struct{} // Singleton class

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) String() string          { return Domain.String() }

type Object *model.Sample

type Query struct {
	PromQL string // `json:",omitempty"`
}

func (q *Query) String() string        { return q.PromQL }
func (q *Query) Class() korrel8r.Class { return Class{} }

type Store struct{ api promv1.API }

func NewStore(base *url.URL, hc *http.Client) (korrel8r.Store, error) {
	c, err := api.NewClient(api.Config{Address: base.String(), Client: hc})
	if err != nil {
		return nil, err
	}
	return &Store{promv1.NewAPI(c)}, nil
}

func (s *Store) Domain() korrel8r.Domain { return Domain }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	value, _, err := s.api.Query(ctx, q.PromQL, time.Now())
	if err != nil {
		return err
	}
	if values, ok := value.(model.Vector); ok {
		for _, v := range values {
			result.Append(v)
		}
	} else {
		result.Append(value)
	}
	return nil
}

func NewOpenshiftStore(ctx context.Context, c client.Client, cfg *rest.Config) (korrel8r.Store, error) {
	host, err := openshift.RouteHost(ctx, c, openshift.ThanosQuerierNSName)
	if err != nil {
		return nil, err
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	return NewStore(&url.URL{Scheme: "https", Host: host}, hc)
}
