// package alert implements korrel8r interfaces on prometheus alerts.
package alert

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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
	_ korrel8r.Query  = &Query{}
	_ korrel8r.Store  = &Store{}
)

var Domain = domain{}

type domain struct{}

func (domain) String() string              { return "alert" }
func (domain) Class(string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class   { return []korrel8r.Class{Class{}} }
func (domain) UnmarshalQuery(r []byte) (korrel8r.Query, error) {
	return impl.UnmarshalQuery(r, &Query{})
}

type Class struct{} // Only one class - "alert"

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) String() string          { return "alert" }
func (c Class) New() korrel8r.Object    { return &models.GettableAlert{} }
func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(*models.GettableAlert); o != nil {
		return o.Labels[alertname]
	}
	return nil
}

const alertname = "alertname"

type Object *models.GettableAlert

type Query struct{ Labels map[string]string }

func (q *Query) Class() korrel8r.Class { return Class{} }

func (domain) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	m := map[string]string{}
	uq := u.Query()
	for k := range uq {
		m[k] = uq.Get(k)
	}
	return &Query{Labels: m}, nil
}

func (domain) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return nil, err
	}
	uq := url.Values{}
	for k, v := range q.Labels {
		uq.Add(k, v)
	}
	return &url.URL{
		Path:     "/monitoring/alerts",
		RawQuery: uq.Encode(),
	}, nil
}

type Store struct {
	manager *client.AlertmanagerAPI
	base    *url.URL
}

func NewStore(u *url.URL, hc *http.Client) (*Store, error) {
	transport := openapiclient.NewWithClient(u.Host, client.DefaultBasePath, []string{u.Scheme}, hc)

	// Append the "/api/v2" path if not already present.
	path, err := url.JoinPath(strings.TrimSuffix(u.Path, client.DefaultBasePath), client.DefaultBasePath)
	if err != nil {
		return nil, err
	}
	u.Path = path

	return &Store{
		manager: client.New(transport, strfmt.Default),
		base:    u,
	}, nil
}

func (Store) Domain() korrel8r.Domain { return Domain }

func (s Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	// See https://petstore.swagger.io/?url=https://raw.githubusercontent.com/prometheus/alertmanager/master/api/v2/openapi.yaml

	var filters []string
	for k, v := range q.Labels {
		filters = append(filters, fmt.Sprintf("%v=%v", k, v))
	}
	resp, err := s.manager.Alert.GetAlerts(alert.NewGetAlertsParamsWithContext(ctx).WithFilter(filters))
	if err != nil {
		return err
	}
	for _, a := range resp.Payload {
		result.Append(a)
	}
	return nil
}
