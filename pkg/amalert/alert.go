// package amalert implements korrel8 interfaces on prometheus amalerts.
package amalert

import (
	"context"
	"fmt"
	"net/http"

	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/common/model"
)

var Domain = domain{}

type domain struct{}

func (d domain) String() string                     { return "alert" }
func (d domain) Class(string) korrel8.Class         { return Class{} }
func (d domain) Classes() []korrel8.Class           { return []korrel8.Class{Class{}} }
func (d domain) Formatter(string) korrel8.Formatter { return nil }

var _ korrel8.Domain = Domain

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) String() string         { return Domain.String() }
func (c Class) New() korrel8.Object    { return &models.GettableAlert{} }

func (c Class) Key(o korrel8.Object) any { return o.(*models.GettableAlert).Labels["alertname"] }

type Object *models.GettableAlert

type Store struct{ manager *client.Alertmanager }

func NewStore(host string, hc *http.Client) *Store {
	transport := openapiclient.NewWithClient(host, client.DefaultBasePath, []string{"https"}, hc)
	return &Store{manager: client.New(transport, strfmt.Default)}
}

// Get implements the korrel8.Store interface.
// The query URL query= parameter is a PromQL label matcher expression with the wrapping
// `{` and `}` being optional, e.g.  `namespace="default",pod=~"myapp-.+"`.
func (s Store) Get(ctx context.Context, query *korrel8.Query, result korrel8.Result) error {
	// TODO: allow to filter on alert state (pending/firing)?
	// TODO: support sorting order (e.g. most recent/oldest, severity)?
	// TODO: allow grouping (all alerts related to podX grouped together)?
	promQL := query.Query().Get("query")
	matchers, err := labels.ParseMatchers(promQL)
	if err != nil {
		return fmt.Errorf("invalid query: %w: %q", err, query)
	}
	if err != nil {
	}

	params := alert.NewGetAlertsParamsWithContext(context.Background())
	resp, err := s.manager.Alert.GetAlerts(params)
	if err != nil {
		return err
	}
	for _, a := range resp.Payload {
		// Convert LabelSet between libraries
		labelSet := model.LabelSet{}
		for k, v := range a.Labels {
			labelSet[model.LabelName(k)] = model.LabelValue(v)
		}
		if labels.Matchers(matchers).Matches((labelSet)) {
			result.Append(a)
		}
	}
	return nil
}
