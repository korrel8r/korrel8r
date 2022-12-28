// package alert implements korrel8 interfaces on prometheus alerts.
package alert

import (
	"context"
	"fmt"
	"net/http"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var Domain korrel8.Domain = domain{}

type domain struct{}

func (d domain) String() string             { return "alert" }
func (d domain) Class(string) korrel8.Class { return Class{} }
func (d domain) Classes() []korrel8.Class   { return []korrel8.Class{Class{}} }

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain   { return Domain }
func (c Class) String() string           { return Domain.String() }
func (c Class) New() korrel8.Object      { return &v1.Alert{} }
func (c Class) Key(o korrel8.Object) any { return o.(Object).Labels["alertname"] }

var _ korrel8.Class = Class{}

type Object = *v1.Alert

type Store struct {
	api  v1.API
	host string
}

func NewStore(host string, rt http.RoundTripper) (*Store, error) {
	host = fmt.Sprintf("https://%s", host)
	client, err := api.NewClient(api.Config{
		Address:      host,
		RoundTripper: rt,
	})
	if err != nil {
		return nil, err
	}
	return &Store{
		api:  v1.NewAPI(client),
		host: host,
	}, nil
}

// Get implements the korrel8.Store interface.
// The  reference "query" parameter is a PromQL label matcher expression with the wrapping
// `{` and `}` being optional, e.g.  `namespace="default",pod=~"myapp-.+"`.
func (s Store) Get(ctx context.Context, ref uri.Reference, result korrel8.Result) error {
	// TODO: allow to filter on alert state (pending/firing)?
	// TODO: support sorting order (e.g. most recent/oldest, severity)?
	// TODO: allow grouping (all alerts related to podX grouped together)?
	promQL := ref.Values().Get("query")
	matchers, err := labels.ParseMatchers(promQL)
	if err != nil {
		return fmt.Errorf("%v: %w: %v", Domain, err, ref)
	}
	resp, err := s.api.Alerts(ctx)
	if err != nil {
		return fmt.Errorf("%v: %w (%v)", Domain, err, s.host)
	}

	for _, a := range resp.Alerts {
		if labels.Matchers(matchers).Matches(a.Labels) {
			result.Append(a)
		}
	}
	return nil
}
