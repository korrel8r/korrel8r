// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/prometheus"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	name = "metric"
)

//go:embed doc.md
var description string

var (
	Domain = &domain{Domain: impl.NewDomain(name, description, Class{})}

	log = logging.Log()

	_ = impl.AssertDomainTypes(Domain, Object{}, Class{}, Query(""), &Store{})
)

type domain struct{ *impl.Domain }

func (d domain) Query(s string) (korrel8r.Query, error) {
	_, qs, err := impl.ParseQuery(d, s)
	return Query(qs), err
}

const StoreKeyMetricURL = name

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	return NewStore(cs[StoreKeyMetricURL], hc)
}

type Class struct{} // Singleton class

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return name }
func (c Class) String() string          { return korrel8r.ClassString(c) }

func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) Preview(o korrel8r.Object) string            { return Preview(o) }
func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(Object); o != nil {
		return o.Fingerprint()
	}
	return nil
}

type Object = model.Metric

func Preview(o korrel8r.Object) string {
	return impl.Preview(o, func(o Object) string { return o.String() })
}

type Store struct {
	*http.Client
	baseURL        *url.URL      // Original URL from configuration
	configuredPort string        // Port from configuration (e.g., "9091")
	k8sClient      client.Client // For RBAC permission checks
	*impl.Store
}

func NewStore(baseURL string, hc *http.Client) (korrel8r.Store, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Get k8s client for RBAC checks
	k8sClient, err := k8s.NewClient(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	return &Store{
		Client:         hc,
		baseURL:        u,
		configuredPort: u.Port(),
		k8sClient:      k8sClient,
		Store:          impl.NewStore(Domain),
	}, nil
}

func (s *Store) Domain() korrel8r.Domain { return Domain }

type response struct {
	Status string `json:"status"`
	Data   []model.Metric
}

func (s *Store) Get(ctx context.Context, kquery korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	query, err := impl.TypeAssert[Query](kquery)
	if err != nil {
		return err
	}

	baseURL := prometheus.EffectiveURL(ctx, s.baseURL, s.k8sClient).JoinPath("/api/v1")

	// NOTE: Store does not use github.com/prometheus/client_golang because the current version v1.19.1
	// does not allow setting the "limit" query parameter. Hand code the REST query.
	q := url.Values{}
	selectors, err := query.Selectors()
	if err != nil {
		return err
	}

	// Extract namespace from selectors for port 9092 requirement
	// Port 9092 (tenancy port) requires namespace query parameter
	namespaces := extractNamespaces(selectors)

	for _, selector := range selectors {
		q.Add("match[]", selector)
	}

	// Add namespace parameter for port 9092 (tenancy port)
	// The prom-label-proxy requires this parameter for namespace scoping
	prometheus.AddNamespaceParams(q, namespaces)

	if c != nil {
		if c.Start != nil && !c.Start.IsZero() {
			q.Set("start", formatTime(*c.Start))
		}
		if c.End != nil && !c.End.IsZero() {
			q.Set("end", formatTime(*c.End))
		}
		if c.Limit != nil && *c.Limit > 0 {
			q.Set("limit", strconv.Itoa(*c.Limit))
		}
	}
	u := baseURL.JoinPath("series")
	u.RawQuery = q.Encode()
	log.V(5).Info("querying tenancy metric", "query", query, "namespaces", namespaces, "url", u.String())
	var r response
	if err := impl.Get(ctx, u, s.Client, &r); err != nil {
		return fmt.Errorf("metric tenancy query error: %w", err)
	}
	if r.Status != "success" {
		return fmt.Errorf("GET %v: unexpected status: %v", u, r.Status)
	}
	for _, m := range r.Data {
		result.Append(m)
	}
	return nil
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}

// extractNamespaces parses metric selectors to extract namespace label values.
// This is needed for port 9092 (tenancy port) which requires namespace query parameters.
func extractNamespaces(selectors []string) map[string]bool {
	namespaces := make(map[string]bool)
	for _, selector := range selectors {
		// Parse the selector to find namespace label
		// Example: kube_pod_info{namespace="developer-namespace"} -> developer-namespace
		// We'll use a simple string matching approach since these are already parsed selectors
		if idx := strings.Index(selector, `namespace="`); idx != -1 {
			start := idx + len(`namespace="`)
			end := strings.Index(selector[start:], `"`)
			if end != -1 {
				ns := selector[start : start+end]
				namespaces[ns] = true
			}
		} else if idx := strings.Index(selector, `namespace='`); idx != -1 {
			start := idx + len(`namespace='`)
			end := strings.Index(selector[start:], `'`)
			if end != -1 {
				ns := selector[start : start+end]
				namespaces[ns] = true
			}
		}
	}
	return namespaces
}
