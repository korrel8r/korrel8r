// package alert implements korrel8r interfaces on prometheus alerts.
package alert

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var (
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query{}
	_ korrel8r.Store  = &Store{}
)

var Domain = domain{}

type domain struct{}

func (domain) String() string                      { return "alert" }
func (domain) Class(string) korrel8r.Class         { return Class{} }
func (domain) Classes() []korrel8r.Class           { return []korrel8r.Class{Class{}} }
func (domain) Query(korrel8r.Class) korrel8r.Query { return &Query{} }

func (domain) ConsoleURLTo(u *url.URL) (korrel8r.Query, error) {
	return Query{PromQL: fmt.Sprintf(`{alertname=%q}`, u.Query().Get("alertname"))}, nil
}

func (domain) QueryToConsoleURL(q korrel8r.Query) (*url.URL, error) {
	return nil, fmt.Errorf("alert query to conosle URL not implemented: %v", q)
}

// TODO consider separating classes by alertname, determines map schema.
// Still would need wildcard class for all alerts.

type Class struct{} // Only one class - "alert"

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) String() string          { return "alert" }
func (c Class) New() korrel8r.Object    { return &models.GettableAlert{} }
func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(*models.GettableAlert); o != nil {
		return o.Labels["alertname"]
	}
	return nil
}

type Object *models.GettableAlert

type Query struct {
	PromQL string
}

func (q Query) String() string        { return q.PromQL }
func (q Query) Class() korrel8r.Class { return Class{} }

type Store struct {
	manager *client.AlertmanagerAPI
	base    *url.URL
}

func NewStore(host string, hc *http.Client) *Store {
	transport := openapiclient.NewWithClient(host, client.DefaultBasePath, []string{"https"}, hc)
	return &Store{
		manager: client.New(transport, strfmt.Default),
		base:    &url.URL{Scheme: "https", Host: host, Path: client.DefaultBasePath},
	}
}

func (Store) Domain() korrel8r.Domain { return Domain }

func (s Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	// See https://petstore.swagger.io/?url=https://raw.githubusercontent.com/prometheus/alertmanager/master/api/v2/openapi.yaml

	params := alert.NewGetAlertsParamsWithContext(ctx)
	if q.PromQL != "" {
		params.WithFilter([]string{q.PromQL})
	}
	resp, err := s.manager.Alert.GetAlerts(params)
	if err != nil {
		return err
	}
	for _, a := range resp.Payload {
		result.Append(a)
	}
	return nil
}
