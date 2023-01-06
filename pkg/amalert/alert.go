// package alert implements korrel8 interfaces on prometheus alerts.
package alert

import (
	"context"
	"net/http"
	"net/url"

	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var Domain = domain{}

type domain struct{}

func (d domain) String() string             { return "alert" }
func (d domain) Class(string) korrel8.Class { return Class{} }
func (d domain) Classes() []korrel8.Class   { return []korrel8.Class{Class{}} }

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) String() string         { return Domain.String() }
func (c Class) New() korrel8.Object    { return &models.GettableAlert{} }
func (c Class) ID(o korrel8.Object) any {
	if o, _ := o.(*models.GettableAlert); o != nil {
		return o.Labels["alertname"]
	}
	return nil
}

type Object *models.GettableAlert

type Store struct {
	manager *client.Alertmanager
	base    *url.URL
}

func NewStore(host string, hc *http.Client) *Store {
	transport := openapiclient.NewWithClient(host, client.DefaultBasePath, []string{"https"}, hc)
	return &Store{
		manager: client.New(transport, strfmt.Default),
		base:    &url.URL{Scheme: "https", Host: host, Path: client.DefaultBasePath},
	}
}

func (s *Store) Resolve(ref uri.Reference) *url.URL { return ref.Resolve(s.base) }

// Get alerts for alertmanager URI reference, see:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/prometheus/alertmanager/master/api/v2/openapi.yaml
func (s Store) Get(ctx context.Context, ref uri.Reference, result korrel8.Appender) error {
	params := alert.NewGetAlertsParamsWithContext(ctx)

	if f := ref.Query().Get("filter"); f != "" {
		params.WithFilter([]string{f})
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
