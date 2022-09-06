// package prometheus Work In Progress
package prometheus

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alanconway/korrel8/pkg/korrel8"
	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
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

func (s AlertStore) Execute(_ context.Context, query korrel8.Query) ([]any, error) {
	// FIXME need to figure out the class hierarchy for metrics & alerts.
	// This is just a placeholder that returns alerts.
	switch query {
	case "alert":
		resp, err := s.Alert.GetAlerts(nil)
		if err != nil {
			return nil, err
		}
		var result []any
		for _, a := range resp.Payload {
			result = append(result, a)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid query for %v: %v", Domain, query)
	}
}
