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
	"github.com/korrel8r/korrel8r/pkg/uri"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
)

var (
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Store  = &Store{}
)

var Domain = domain{}

type domain struct{}

func (domain) String() string                                    { return "alert" }
func (domain) Class(string) korrel8r.Class                        { return Class{} }
func (domain) Classes() []korrel8r.Class                          { return []korrel8r.Class{Class{}} }
func (domain) RefClass(ref uri.Reference) (korrel8r.Class, error) { return Class{}, nil }

func (domain) RefConsoleToStore(ref uri.Reference) (korrel8r.Class, uri.Reference, error) {
	return Class{}, MakeRef(ref.Query().Get("alertname")), nil
}

func MakeRef(alertname string) uri.Reference {
	filter := fmt.Sprintf(`alertname="%v"`, alertname)
	query := uri.Values{"filter": []string{filter}}
	return uri.Reference{Path: "/api/v1/alerts", RawQuery: query.Encode()}
}

func (domain) RefStoreToConsole(class korrel8r.Class, ref uri.Reference) (uri.Reference, error) {
	return uri.Reference{}, fmt.Errorf("alert store URI to conosle not implemented: %v", ref)
}

type Class struct{} // Only one class

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) String() string         { return "alert" }
func (c Class) New() korrel8r.Object    { return &models.GettableAlert{} }
func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(*models.GettableAlert); o != nil {
		return o.Labels["alertname"]
	}
	return nil
}

var _ korrel8r.Class = Class{}

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
func (s Store) Get(ctx context.Context, ref uri.Reference, result korrel8r.Appender) error {
	params := alert.NewGetAlertsParamsWithContext(ctx)

	if f := ref.Query()["filter"]; len(f) > 0 {
		params.WithFilter(f)
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

func (Store) RefStoreToConsole(class korrel8r.Class, ref uri.Reference) (uri.Reference, error) {
	return Domain.RefStoreToConsole(class, ref)
}

func (Store) RefConsoleToStore(ref uri.Reference) (korrel8r.Class, uri.Reference, error) {
	return Domain.RefConsoleToStore(ref)
}

func (Store) RefClass(ref uri.Reference) (korrel8r.Class, error) { return Domain.RefClass(ref) }

var _ korrel8r.RefConverter = &Store{}
