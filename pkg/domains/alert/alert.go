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
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	openapiclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
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
	alertmanagerAPI *client.AlertmanagerAPI
	prometheusAPI   v1.API
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

	return &Store{
		alertmanagerAPI: alertmanagerAPI,
		prometheusAPI:   prometheusAPI,
		Store:           impl.NewStore(Domain),
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

func (s *Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	// Gather matching alerts from the Prometheus Rules API.
	prometheusRules, err := s.prometheusAPI.Rules(ctx)
	if err != nil {
		return fmt.Errorf("failed to query rules from Prometheus API: %w", err)
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
	var filters []string
	for k, v := range subQuery {
		filters = append(filters, fmt.Sprintf("%v=%v", k, v))
	}
	alertManagerAlerts, err := s.alertmanagerAPI.Alert.GetAlerts(alert.NewGetAlertsParamsWithContext(ctx).WithFilter(filters))
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts from Alertmanager API: %w", err)
	}
	for _, pa := range alerts {
		s.augmentAlert(pa, alertManagerAlerts)
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
