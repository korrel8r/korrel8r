// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
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
	if err != nil {
		return nil, err
	}
	// Reject a malformed ?namespace= suffix now. PromQL validity is deferred to
	// Query.Selectors, which parses the expression when the query is first used.
	if _, _, err := splitQuery(qs); errors.Is(err, errSuffix) {
		return nil, err
	}
	return Query(qs), nil
}

const (
	StoreKeyMetricURL     = name
	StoreKeyNamespacedURL = "namespaced"
)

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	return NewStore(cs[StoreKeyMetricURL], cs[StoreKeyNamespacedURL], hc)
}

func (o Object) String() string {
	metricName := o.Labels["__name__"]
	labelStrings := make([]string, 0, len(o.Labels))
	for k, v := range o.Labels {
		if k != "__name__" {
			labelStrings = append(labelStrings, fmt.Sprintf("%s=%q", k, v))
		}
	}
	sort.Strings(labelStrings)
	if len(labelStrings) == 0 {
		if metricName != "" {
			return metricName
		}
		return "{}"
	}
	return fmt.Sprintf("%s{%s}", metricName, strings.Join(labelStrings, ", "))
}

type Class struct{} // Singleton class

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return name }
func (c Class) String() string          { return korrel8r.ClassString(c) }

func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) Preview(o korrel8r.Object) string            { return Preview(o) }
func (c Class) ID(o korrel8r.Object) any {
	if o, ok := o.(Object); ok {
		return o.Fingerprint
	}
	return nil
}

type Object struct {
	Labels      map[string]string `json:"labels"`
	Fingerprint string            `json:"fingerprint"`
}

func convertMetricToMap(m model.Metric) map[string]string {
	res := make(map[string]string, len(m))
	for k, v := range m {
		res[string(k)] = string(v)
	}
	return res
}

func Preview(o korrel8r.Object) string {
	return impl.Preview(o, func(o Object) string { return o.String() })
}

type Store struct {
	*http.Client
	baseURL       *url.URL // Cluster-scoped URL from configuration
	namespacedURL *url.URL // Namespace-scoped URL from configuration
	*impl.Store
}

func NewStore(baseURL, namespacedURL string, hc *http.Client) (korrel8r.Store, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("metric URL: %w", err)
	}
	var nu *url.URL
	if namespacedURL != "" {
		nu, err = url.Parse(namespacedURL)
		if err != nil {
			return nil, fmt.Errorf("namespaced metric URL: %w", err)
		}
	}

	return &Store{
		Client:        hc,
		baseURL:       u,
		namespacedURL: nu,
		Store:         impl.NewStore(Domain),
	}, nil
}

func (s *Store) Domain() korrel8r.Domain { return Domain }

type response struct {
	Status string `json:"status"`
	Data   []model.Metric
}

func (s *Store) Get(ctx context.Context, kquery korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	// Type assertion errors are not store errors, treat as "not found".
	query, ok := kquery.(Query)
	if !ok {
		return nil
	}

	params, err := query.Params() // Validate a suffix built without Domain.Query.
	if err != nil {
		return err
	}
	selectors, err := query.Selectors()
	if err != nil {
		return err
	}
	baseURL, namespaces := s.queryTarget(params)
	baseURL = baseURL.JoinPath("/api/v1")

	// NOTE: Store does not use github.com/prometheus/client_golang because the current version v1.19.1
	// does not allow setting the "limit" query parameter. Hand code the REST query.
	q := url.Values{}
	for _, selector := range selectors {
		q.Add("match[]", selector)
	}

	// Only explicit suffix namespaces are sent to the namespace-scoped endpoint.
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
	log.V(5).Info("querying metric", "query", query, "namespaces", namespaces, "url", u.String())
	var r response
	if err := impl.Get(ctx, u, s.Client, &r); err != nil {
		return fmt.Errorf("metric query error: %w", err)
	}
	if r.Status != "success" {
		return fmt.Errorf("GET %v: unexpected status: %v", u, r.Status)
	}
	for _, m := range r.Data {
		result.Append(Object{
			Labels:      convertMetricToMap(m),
			Fingerprint: m.Fingerprint().String(),
		})
	}
	return nil
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}

// queryTarget selects the namespace-scoped URL for explicit namespace suffixes when configured.
func (s *Store) queryTarget(params url.Values) (*url.URL, map[string]bool) {
	if ns := params[namespaceParam]; len(ns) > 0 {
		namespaces := make(map[string]bool, len(ns))
		for _, n := range ns {
			namespaces[n] = true
		}
		if s.namespacedURL != nil {
			return s.namespacedURL, namespaces
		}
		return s.baseURL, namespaces
	}
	return s.baseURL, nil
}
