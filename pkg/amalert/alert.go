// package amalert implements korrel8 interfaces on prometheus amalerts.
package amalert

import (
	"context"
	"net/http"
	"net/url"

	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var Domain = domain{}

type domain struct{}

func (d domain) String() string                { return "alert" }
func (d domain) Class(string) korrel8.Class    { return Class{} }
func (d domain) KnownClasses() []korrel8.Class { panic("FIXME") }
func (d domain) NewQuery() korrel8.Query       { return &Query{} }

var _ korrel8.Domain = Domain

type Store struct {
	manager *client.Alertmanager
}

func NewStore(host string, hc *http.Client) *Store {
	transport := openapiclient.NewWithClient(host, client.DefaultBasePath, []string{"https"}, hc)
	return &Store{manager: client.New(transport, strfmt.Default)}
}

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain         { return Domain }
func (c Class) String() string                 { return Domain.String() }
func (c Class) New() korrel8.Object            { return &models.GettableAlert{} }
func (c Class) Key(o korrel8.Object) any       { return o }
func (c Class) Contains(o korrel8.Object) bool { _, ok := o.(Object); return ok }

type Query struct{} // FIXME - gets all alerts.

func (q *Query) String() string                 { return Domain.String() }
func (q *Query) Browser(base *url.URL) *url.URL { panic("FIXME") }
func (q *Query) REST(base *url.URL) *url.URL    { panic("FIXME") }

type Object *models.GettableAlert

func (s Store) Get(ctx context.Context, q korrel8.Query, result korrel8.Result) error {
	params := alert.NewGetAlertsParamsWithContext(context.Background())
	resp, err := s.manager.Alert.GetAlerts(params)
	if err != nil {
		return err
	}
	for _, a := range resp.Payload {
		result.Append(a)
	}
	return nil
}
