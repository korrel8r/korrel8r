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
)

var Domain = domain{}

type domain struct{}

func (domain) Name() string                { return "alert" }
func (d domain) String() string            { return d.Name() }
func (domain) Description() string         { return "Alerts that metric values are out of bounds." }
func (domain) Class(string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class   { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	var q Query
	_, err := impl.UnmarshalQueryString(d, s, &q)
	return q, err
}

const (
	StoreKeyMetrics      = "metrics"
	StoreKeyAlertmanager = "alertmanager"
)

func (domain) Store(sc korrel8r.StoreConfig) (korrel8r.Store, error) {
	metrics, alertmanager := sc[StoreKeyMetrics], sc[StoreKeyAlertmanager]
	metricsURL, err := url.Parse(metrics)
	if err != nil {
		return nil, err
	}
	alertmanagerURL, err := url.Parse(alertmanager)
	if err != nil {
		return nil, err
	}
	return NewStore(alertmanagerURL, metricsURL, nil)
}

// Class is represents any Prometheus alert. There is only a single class, named "alert".
type Class struct{}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return "alert" }
func (c Class) String() string          { return korrel8r.ClassName(c) }
func (c Class) Description() string {
	return "An indication that some collection of metrics is outside of expected values."
}
func (c Class) New() korrel8r.Object { return &Object{} }
func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(*Object); o != nil {
		// The identity of an alert is defined by its labels.
		return o.Fingerprint
	}
	return nil
}
func (c Class) Preview(o korrel8r.Object) string {
	if o, _ := o.(*Object); o != nil {
		return o.Labels["alertname"]
	}
	return ""
}

// Object is represented by the JSON serialization the following type.
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
type Query map[string]string

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Query() string         { return korrel8r.JSONString(q) }
func (q Query) String() string        { return korrel8r.QueryName(q) }

func (domain) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	m := map[string]string{}
	uq := u.Query()
	for k := range uq {
		m[k] = uq.Get(k)
	}
	return Query(m), nil
}

func (domain) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return nil, err
	}
	uq := url.Values{
		"rowFilter-alert-state": []string{""}, // do not filter by alert state.
	}
	alertFilter := make([]string, 0, len(q))
	for k, v := range q {
		alertFilter = append(alertFilter, fmt.Sprintf("%s=%s", k, v))
	}
	uq.Add("alerts", strings.Join(alertFilter, ","))

	return &url.URL{
		Path:     "/monitoring/alerts",
		RawQuery: uq.Encode(),
	}, nil
}

// Store is a client of Prometheus and AlertManager.
type Store struct {
	alertmanagerAPI *client.AlertmanagerAPI
	prometheusAPI   v1.API
}

// NewStore creates a new store client for a Prometheus URL.
func NewStore(alertmanagerURL *url.URL, prometheusURL *url.URL, hc *http.Client) (korrel8r.Store, error) {
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

func (Store) Domain() korrel8r.Domain { return Domain }

func convertLabelSetToMap(m model.LabelSet) map[string]string {
	res := make(map[string]string, len(m))
	for k, v := range m {
		res[string(k)] = string(v)
	}

	return res
}

// matches returns true if the Prometheus alert matches the korrel8r query.
func (q Query) matches(a *v1.Alert) bool {
	for k, v := range q {
		v2 := string(a.Labels[model.LabelName(k)])
		if v != v2 {
			return false
		}
	}

	return true
}

func (s Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	// Gather matching alerts from the Prometheus Rules API.
	rulesResult, err := s.prometheusAPI.Rules(ctx)
	if err != nil {
		return fmt.Errorf("failed to query rules from Prometheus API: %w", err)
	}

	var alerts = []*Object{}
	for _, rg := range rulesResult.Groups {
		for _, r := range rg.Rules {
			ar, ok := r.(v1.AlertingRule)
			if !ok {
				continue
			}

			for _, a := range ar.Alerts {
				if !q.matches(a) {
					continue
				}

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

	// Gather matching alerts from the Alertmanager API and merge with the existing alerts.
	var filters []string
	for k, v := range q {
		filters = append(filters, fmt.Sprintf("%v=%v", k, v))
	}
	resp, err := s.alertmanagerAPI.Alert.GetAlerts(alert.NewGetAlertsParamsWithContext(ctx).WithFilter(filters))
	if err != nil {
		return fmt.Errorf("failed to query alerts from Alertmanager API: %w", err)
	}

	for _, a := range resp.Payload {
		// We can't perform an exact label comparison because alerts from
		// Alertmanager may have more labels than Prometheus alerts (due to
		// external labels for instance).
		// We consider an Alertmanager alert to be the same as a Prometheus
		// alert if the Alertmanager labels are a super-set of the Prometheus
		// labels.
		var o *Object
		for _, pa := range alerts {
			found := true
			for k, v := range pa.Labels {
				if a.Labels[k] != v {
					found = false
					break
				}
			}

			if found {
				o = pa
				break
			}
		}

		// If the alert doesn't exist in Prometheus, skip it.
		if o == nil {
			break
		}

		o.StartsAt = time.Time(*a.StartsAt)
		o.EndsAt = time.Time(*a.EndsAt)
		o.GeneratorURL = a.Alert.GeneratorURL.String()
		for _, r := range a.Receivers {
			o.Receivers = append(o.Receivers, Receiver{Name: *r.Name})
		}
		o.SilencedBy = a.Status.SilencedBy
		o.InhibitedBy = a.Status.InhibitedBy

		if o.Status == "" {
			o.Status = *a.Status.State
			if o.Status != "suppressed" {
				o.Status = "firing"
			}
		} else if *a.Status.State == "suppressed" {
			o.Status = *a.Status.State
		}
	}

	for _, a := range alerts {
		result.Append(a)
	}

	return nil
}
