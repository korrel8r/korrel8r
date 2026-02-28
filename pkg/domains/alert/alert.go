// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package alert provides Prometheus alerts, queries and access to Thanos and AlertManager stores.
//
// # Class
//
// There is a single class `alert:alert`.
//
// # Object
//
// An alert object is represented by this Go type.
// Rules starting from an alert should use the capitalized Go field names rather than the lowercase JSON names.
// [Object]
//
// # Query
//
// A JSON map of string names to string values, matched against alert labels, for example:
//
//	alert:alert:{"alertname":"KubeStatefulSetReplicasMismatch","container":"kube-rbac-proxy-main","namespace":"openshift-logging"}
//
// To query mutiple alerts at the same time, it's possible to provide an array of maps:
//
//	alert:alert:[{"alertname":"alert1"},{"alertname":"alert2"}]
//
// # Store
//
// A client of Prometheus and/or AlertManager. Store configuration:
//
//	domain: alert
//	metrics: PROMETHEUS_URL
//	alertmanager: ALERTMANAGER_URL
//
// Either or both of `metrics` or `alertmanager` may be present.
package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logging.Log()

	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query{}
	_ korrel8r.Store  = &Store{}
	_ korrel8r.Object = &Object{}
)

var Domain = domain{Domain: impl.NewDomain("alert", "Alerts that metric values are out of bounds.", Class{})}

type domain struct{ *impl.Domain }

func (d domain) Query(s string) (korrel8r.Query, error) {
	var query []map[string]string
	_, qs, err := impl.ParseQuery(d, s)
	if err != nil {
		return nil, err
	}

	var simpleQuery map[string]string
	err = impl.Unmarshal([]byte(qs), &simpleQuery)
	if err == nil {
		query = []map[string]string{simpleQuery}
	} else {
		// Try more complex variant of the query, accepting list of maps.
		err = impl.Unmarshal([]byte(qs), &query)
		if err != nil {
			return nil, err
		}
	}

	return Query{Qs: qs, Parsed: query}, nil
}

const (
	StoreKeyMetrics      = "metrics"
	StoreKeyAlertmanager = "alertmanager"
)

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	metrics, alertmanager := cs[StoreKeyMetrics], cs[StoreKeyAlertmanager]
	metricsURL, err := url.Parse(metrics)
	if err != nil {
		return nil, err
	}
	alertmanagerURL, err := url.Parse(alertmanager)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	return NewStore(alertmanagerURL, metricsURL, hc)
}

// Class is represents any Prometheus alert. There is only a single class, named "alert".
type Class struct{}

func (c Class) Domain() korrel8r.Domain                     { return Domain }
func (c Class) Name() string                                { return "alert" }
func (c Class) String() string                              { return korrel8r.ClassString(c) }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[*Object](b) }
func (c Class) ID(o korrel8r.Object) any {
	if o, ok := o.(*Object); ok {
		return o.Fingerprint
	}
	return nil
}

func (c Class) Preview(o korrel8r.Object) string {
	if o, ok := o.(*Object); ok {
		return o.Labels["alertname"]
	}
	return ""
}

// Object contains alert data, passed as *Object when used as a korrel8r.Object.
type Object struct {
	// Common fields.
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Status      string            `json:"status"` // inactive|pending|firing|suppressed
	StartsAt    time.Time         `json:"startsAt"`

	// Prometheus fields.
	Value       string `json:"value"`
	Expression  string `json:"expression"`
	Fingerprint string `json:"fingerprint"`

	// Alertmanager fields.
	EndsAt       time.Time  `json:"endsAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Receivers    []Receiver `json:"receivers"`
	InhibitedBy  []string   `json:"inhibitedBy"`
	SilencedBy   []string   `json:"silencedBy"`
	GeneratorURL string     `json:"generatorURL"`
}

// Receiver is a named receiver, part of Object.
type Receiver struct {
	Name string `json:"name"`
}

// Query is a map of label name:value pairs for matching alerts, serialized as JSON.
type Query struct {
	Qs     string
	Parsed []map[string]string
}

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return q.Qs }
func (q Query) String() string        { return korrel8r.QueryString(q) }

// Store is a client of Prometheus and AlertManager.
type Store struct {
	alertmanagerAPI      *client.AlertmanagerAPI
	prometheusAPI        v1.API
	prometheusURL        *url.URL         // Original URL from configuration
	prometheusConfigPort string           // Port from configuration (e.g., "9091")
	httpClient           *http.Client     // HTTP client for recreating prometheus client
	k8sClient            k8sclient.Client // For RBAC permission checks
	*impl.Store
}

// NewStore creates a new store client for a Prometheus URL.
func NewStore(alertmanagerURL *url.URL, prometheusURL *url.URL, hc *http.Client) (*Store, error) {
	alertmanagerAPI, err := newAlertmanagerClient(alertmanagerURL, hc)
	if err != nil {
		return nil, err
	}

	prometheusAPI, err := newPrometheusClient(prometheusURL, hc)
	if err != nil {
		return nil, err
	}

	// Get k8s client for RBAC checks
	k8sClient, err := k8s.NewClient(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	return &Store{
		alertmanagerAPI:      alertmanagerAPI,
		prometheusAPI:        prometheusAPI,
		prometheusURL:        prometheusURL,
		prometheusConfigPort: prometheusURL.Port(),
		httpClient:           hc,
		k8sClient:            k8sClient,
		Store:                impl.NewStore(Domain),
	}, nil
}

func newAlertmanagerClient(u *url.URL, hc *http.Client) (*client.AlertmanagerAPI, error) {
	transport := openapiclient.NewWithClient(u.Host, client.DefaultBasePath, []string{u.Scheme}, hc)

	// Append the "/api/v2" path if not already present.
	path, err := url.JoinPath(strings.TrimSuffix(u.Path, client.DefaultBasePath), client.DefaultBasePath)
	if err != nil {
		return nil, err
	}
	u.Path = path

	return client.New(transport, strfmt.Default), nil
}

func newPrometheusClient(u *url.URL, hc *http.Client) (v1.API, error) {
	client, err := api.NewClient(api.Config{
		Address: u.String(),
		Client:  hc,
	})
	if err != nil {
		return nil, err
	}

	return v1.NewAPI(client), nil
}

func (*Store) Domain() korrel8r.Domain { return Domain }

// getEffectivePrometheusAPI returns a Prometheus API client with the appropriate port based on user permissions.
// Admin users (with cluster-monitoring-view) use the configured port (typically 9091).
// Non-admin users use the tenancy port 9093 for namespace-scoped rules/alerts access.
func (s *Store) getEffectivePrometheusAPI(ctx context.Context) (v1.API, error) {
	u, err := k8s.GetEffectivePrometheusURL(ctx, s.prometheusURL, s.prometheusConfigPort, s.k8sClient, "alert", k8s.TenancyPortRules)
	if err != nil {
		return nil, err
	}
	return newPrometheusClient(u, s.httpClient)
}

func convertLabelSetToMap(m model.LabelSet) map[string]string {
	res := make(map[string]string, len(m))
	for k, v := range m {
		res[string(k)] = string(v)
	}

	return res
}

// matchesSubquery returns true if the prometheus alert matches part of korrel8r query.
func matchesSubquery(q map[string]string, a *v1.Alert) bool {
	for k, v := range q {
		v2 := string(a.Labels[model.LabelName(k)])
		if v != v2 {
			return false
		}
	}

	return true
}

// extractNamespacesFromQuery extracts unique namespace values from parsed alert queries.
func extractNamespacesFromQuery(q Query) map[string]bool {
	namespaces := make(map[string]bool)
	for _, subq := range q.Parsed {
		if ns, ok := subq["namespace"]; ok && ns != "" {
			namespaces[ns] = true
		}
	}
	return namespaces
}

// getRulesWithNamespaceFilter queries the Rules API with namespace filtering.
// This is used for non-admin users on port 9093 which requires namespace parameters.
func (s *Store) getRulesWithNamespaceFilter(ctx context.Context, promAPI v1.API, q Query) (v1.RulesResult, error) {
	// Extract unique namespaces from all subqueries
	namespaces := extractNamespacesFromQuery(q)

	// If no namespaces found in query, return empty result
	// Port 9093 requires namespace filtering
	if len(namespaces) == 0 {
		log.V(2).Info("no namespaces found in alert query, returning empty result")
		return v1.RulesResult{}, nil
	}

	log.V(2).Info("querying rules API with namespace filter", "namespaces", namespaces)

	// Build URL with namespace query parameters
	// Port 9093 expects: /api/v1/rules?namespace=ns1&namespace=ns2
	u, err := k8s.GetEffectivePrometheusURL(ctx, s.prometheusURL, s.prometheusConfigPort, s.k8sClient, "alert", k8s.TenancyPortRules)
	if err != nil {
		return v1.RulesResult{}, err
	}

	// Add /api/v1/rules path
	rulesURL := u.JoinPath("/api/v1/rules")

	// Add namespace query parameters (required by port 9093)
	queryParams := rulesURL.Query()
	k8s.AddNamespaceParams(queryParams, namespaces)
	rulesURL.RawQuery = queryParams.Encode()

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", rulesURL.String(), nil)
	if err != nil {
		return v1.RulesResult{}, fmt.Errorf("failed to create rules request: %w", err)
	}

	log.V(3).Info("executing rules API request", "url", rulesURL.String())
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.V(1).Info("failed to execute rules request", "error", err, "url", rulesURL.String())
		return v1.RulesResult{}, fmt.Errorf("failed to execute rules request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.V(1).Info("failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.V(1).Info("rules request returned non-OK status", "status", resp.StatusCode, "url", rulesURL.String())
		return v1.RulesResult{}, fmt.Errorf("rules request returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp struct {
		Status string         `json:"status"`
		Data   v1.RulesResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return v1.RulesResult{}, fmt.Errorf("failed to decode rules response: %w", err)
	}

	if apiResp.Status != "success" {
		return v1.RulesResult{}, fmt.Errorf("rules API returned status: %s", apiResp.Status)
	}

	return apiResp.Data, nil
}

func (s *Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	// Get the effective Prometheus API based on user permissions (admin vs non-admin)
	promAPI, err := s.getEffectivePrometheusAPI(ctx)
	if err != nil {
		return fmt.Errorf("failed to get effective Prometheus API: %w", err)
	}

	// Check if user has cluster-level access
	hasClusterAccess, err := k8s.CanAccessPrometheusAPI(ctx, s.k8sClient)
	if err != nil {
		hasClusterAccess = false
	}

	var prometheusRules v1.RulesResult
	if hasClusterAccess {
		// Admin users: use the Rules API without namespace filtering (port 9091)
		prometheusRules, err = promAPI.Rules(ctx)
		if err != nil {
			return fmt.Errorf("failed to query rules from Prometheus API: %w", err)
		}
	} else {
		// Non-admin users: use the Rules API with namespace filtering (port 9093)
		prometheusRules, err = s.getRulesWithNamespaceFilter(ctx, promAPI, q)
		if err != nil {
			return fmt.Errorf("failed to query rules with namespace filter: %w", err)
		}
	}

	for _, subquery := range q.Parsed {
		alerts, err := s.getSubquery(ctx, prometheusRules, subquery)
		if err != nil {
			return err
		}

		for _, a := range alerts {
			// Only include alerts that overlap with the constraint interval or have no start/end time.
			if c.CompareTime(a.StartsAt) <= 0 && c.CompareTime(a.EndsAt) >= 0 {
				result.Append(a)
			}
		}
	}

	return nil
}

func (s *Store) getSubquery(ctx context.Context, prometheusRules v1.RulesResult, subQuery map[string]string) ([]*Object, error) {
	var alerts []*Object
	for _, rg := range prometheusRules.Groups {
		for _, r := range rg.Rules {
			ar, ok := r.(v1.AlertingRule)
			if !ok {
				continue
			}
			for _, a := range ar.Alerts {
				if matchesSubquery(subQuery, a) {
					alerts = append(alerts, &Object{
						Labels:      convertLabelSetToMap(a.Labels),
						Annotations: convertLabelSetToMap(a.Annotations),
						Status:      string(a.State),
						Value:       a.Value,
						StartsAt:    a.ActiveAt,
						Expression:  ar.Query,
						Fingerprint: a.Labels.Fingerprint().String(),
					})
				}
			}
		}
	}

	// Gather matching alerts from the Alertmanager API and merge with the existing alerts.
	// This is optional - if the user doesn't have access to Alertmanager, we'll just
	// return alerts from the Rules API without timing information.
	var filters []string
	for k, v := range subQuery {
		filters = append(filters, fmt.Sprintf("%v=%v", k, v))
	}
	alertManagerAlerts, err := s.alertmanagerAPI.Alert.GetAlerts(alert.NewGetAlertsParamsWithContext(ctx).WithFilter(filters))
	if err != nil {
		// Log the error but don't fail - Alertmanager access may be restricted
		// Non-admin users may not have access to Alertmanager
		// Return alerts from Rules API only
	} else {
		// Augment alerts with Alertmanager data if available
		for _, pa := range alerts {
			s.augmentAlert(pa, alertManagerAlerts)
		}
	}
	return alerts, nil

}

// augmentAlert augment a prometheus alert using the matching alertManager alert if there is one.
func (*Store) augmentAlert(pa *Object, alertManagerAlerts *alert.GetAlertsOK) {
	// We can't perform an exact label comparison because alerts from
	// Alertmanager may have more labels than Prometheus alerts (due to
	// external labels for instance).
	// We consider an Alertmanager alert to be the same as a Prometheus
	// alert if the Alertmanager labels are a super-set of the Prometheus
	// labels.
	for _, ama := range alertManagerAlerts.Payload {
		for k, v := range pa.Labels {
			if ama.Labels[k] != v {
				continue
			}
			pa.StartsAt = time.Time(*ama.StartsAt)
			pa.EndsAt = time.Time(*ama.EndsAt)
			pa.GeneratorURL = ama.GeneratorURL.String()
			for _, r := range ama.Receivers {
				pa.Receivers = append(pa.Receivers, Receiver{Name: *r.Name})
			}
			pa.SilencedBy = ama.Status.SilencedBy
			pa.InhibitedBy = ama.Status.InhibitedBy

			if pa.Status == "" {
				pa.Status = *ama.Status.State
				if pa.Status != "suppressed" {
					pa.Status = "firing"
				}
			} else if *ama.Status.State == "suppressed" {
				pa.Status = *ama.Status.State
			}
		}
	}
}
