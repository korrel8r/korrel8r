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

const Domain korrel8.Domain = "alert"

type Store struct {
	manager *client.Alertmanager
}

func NewStore(host string, hc *http.Client) *Store {
	transport := openapiclient.NewWithClient(host, client.DefaultBasePath, []string{"https"}, hc)
	return &Store{manager: client.New(transport, strfmt.Default)}
}

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) String() string         { return string(Domain) }

type Object struct{ *models.GettableAlert }

func (o Object) Identifier() korrel8.Identifier { return o.Labels }
func (o Object) Class() korrel8.Class           { return Class{} }
func (o Object) Native() any                    { return o.GettableAlert }

// Query is a JSON object containing JSON-commpatible fields of
// https://pkg.go.dev/github.com/prometheus/alertmanager/api/v2/client/alert#GetAlertsParams
func (s Store) Query(ctx context.Context, query string) ([]korrel8.Object, error) {
	if query == "" {
		query = "{}" // Allow empty string as empty object
	}
	params := alert.NewGetAlertsParamsWithContext(ctx)
	if err := json.Unmarshal([]byte(query), params); err != nil {
		return nil, err
	}
	resp, err := s.manager.Alert.GetAlerts(params)
	if err != nil {
		return nil, err
	}
	var result []korrel8.Object
	for _, a := range resp.Payload {
		result = append(result, Object{a})
	}
	return result, nil
}
