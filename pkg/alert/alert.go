// package alert implements korrel8 interfaces on prometheus alerts.
package alert

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/alanconway/korrel8/pkg/korrel8"
	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var Domain = domain{}

type domain struct{}

func (d domain) String() string                { return "alert" }
func (d domain) Class(string) korrel8.Class    { return Class{} }
func (d domain) KnownClasses() []korrel8.Class { panic("not implemented") } // FIXME

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

type Object *models.GettableAlert

// FIXME use a REST URI for consistency?

// Query is a JSON object containing JSON-commpatible fields of
// https://pkg.go.dev/github.com/prometheus/alertmanager/api/v2/client/alert#GetAlertsParams
func (s Store) Get(ctx context.Context, query korrel8.Query, result korrel8.Result) error {
	if query == "" {
		query = "{}" // Allow empty string as empty object
	}
	params := alert.NewGetAlertsParamsWithContext(ctx)
	if err := json.Unmarshal([]byte(query), params); err != nil {
		return err
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
