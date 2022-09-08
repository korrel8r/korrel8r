// package prometheus Work In Progress
package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/alanconway/korrel8/pkg/korrel8"
	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// Domain is the prometheus domain name.
const Domain korrel8.Domain = "prometheus"

// AlertStore is a store of alerts.
type AlertStore struct {
	*client.Alertmanager
}

// NewAlertStore returns an alert store using the given host and HTTP client.
func NewAlertStore(host string, hc *http.Client) *AlertStore {
	transport := openapiclient.NewWithClient(host, client.DefaultBasePath, []string{"https"}, hc)
	return &AlertStore{Alertmanager: client.New(transport, strfmt.Default)}
}

type Class struct{ reflect.Type }

func (c Class) Domain() korrel8.Domain { return Domain }
func ClassOf(o any) korrel8.Class      { return Class{reflect.TypeOf(o).Elem()} }

type AlertObject struct{ *models.GettableAlert }

func (o AlertObject) Identifier() korrel8.Identifier { return o.Labels }
func (o AlertObject) Class() korrel8.Class           { return Class{reflect.TypeOf(o)} }

func (s AlertStore) Execute(_ context.Context, query string) ([]korrel8.Object, error) {
	// TODO this just handles Alerts, there are other prometheus API types we need to deal with.
	switch query {
	case "alert":
		resp, err := s.Alert.GetAlerts(nil)
		if err != nil {
			return nil, err
		}
		var result []korrel8.Object
		for _, a := range resp.Payload {
			result = append(result, AlertObject{a})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid query for %v: %v", Domain, query)
	}
}
