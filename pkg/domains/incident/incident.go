// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package incident provides mapping to https://github.com/openshift/cluster-health-analyzer
// incidents.
//
// # Class
//
// There is a single class `incident:incident`.
//
// # Object
//
// An incident object contains id and mapping to the sources (alert is the only
// supported source type at the moment).
//
// # Query
//
// One can query the incident by its id.
//
//	incident:incident:{"id":"id-of-the-incident"}
//
// It's also possible to provide labels from
// an alert to get a corresponding incident.
//
//	incident:incident:{"alertLabels":{
//	 "alertname":"AlertmanagerReceiversNotConfigured",
//	 "namespace":"openshift-monitoring"}}
//
// # Store
//
// A client of Prometheus. Store configuration:
//
//	domain: incident
//	metrics: PROMETHEUS_URL
package incident

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Query  = Query{}
	_ korrel8r.Store  = &Store{}
	_ korrel8r.Object = &Object{}

	log = logging.Log()
)

var Domain = domain{}

type domain struct{}

const (
	name        = "incident"
	description = "Incidents group alerts into higher-level groups."

	srcLabelPrefix = "src_"
)

func (domain) Name() string                { return name }
func (d domain) String() string            { return d.Name() }
func (domain) Description() string         { return description }
func (domain) Class(string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class   { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	_, query, err := impl.UnmarshalQueryString[Query](d, s)
	return query, err
}

const (
	StoreKeyMetrics = "metrics"
)

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	metrics := cs[StoreKeyMetrics]
	metricsURL, err := url.Parse(metrics)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	return NewStore(metricsURL, hc)
}

// Class represents any Incident. There is only a single class, named "incident".
type Class struct{}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return name }
func (c Class) String() string          { return impl.ClassString(c) }
func (c Class) Description() string {
	return description
}
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[*Object](b) }
func (c Class) ID(o korrel8r.Object) any {
	if o, ok := o.(*Object); ok {
		return o.Id
	}
	return nil
}

func (c Class) Preview(o korrel8r.Object) string {
	if o, ok := o.(*Object); ok {
		return o.Id
	}
	return ""
}

// Object contains incident data, passed as *Object when used as a korrel8r.Object.
type Object struct {
	// Common fields.
	Id           string              `json:"id"`
	AlertsLabels []map[string]string `json:"alertsLabels"`

	// Prometheus fields.
	Value string `json:"value"`
}

type Query struct {
	Id string `json:"id,omitempty"`
	// Alert labels to match against
	AlertLabels map[string]string `json:"alertLabels,omitempty"`
}

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { b, _ := json.Marshal(q); return string(b) }
func (q Query) String() string        { return impl.QueryString(q) }

// Store is a client of Prometheus.
type Store struct {
	prometheusAPI v1.API
}

// NewStore creates a new store client for a Prometheus URL.
func NewStore(prometheusURL *url.URL, hc *http.Client) (korrel8r.Store, error) {
	prometheusAPI, err := newPrometheusClient(prometheusURL, hc)
	if err != nil {
		return nil, err
	}

	return &Store{
		prometheusAPI: prometheusAPI,
	}, nil
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

func (s Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	promq, t, err := preparePromQL(q, c)
	if err != nil {
		return err
	}

	log.V(3).Info("Loading the incidents", "query", promq)

	resp, _, err := s.prometheusAPI.Query(ctx, promq, t)
	if err != nil {
		return fmt.Errorf("Failed to query incidents from Prometheus API: %w", err)
	}

	data := resp.(model.Vector)
	log.V(3).Info("Incidents data loaded", "count", len(data))

	incidents := loadObjects(data)
	log.V(3).Info("Incidents loaded", "count", len(incidents))

	incidents = filterObjects(incidents, q)
	log.V(2).Info("Incidents filtered", "count", len(incidents))

	for _, i := range incidents {
		result.Append(i)
	}

	return nil
}

func preparePromQL(q Query, c *korrel8r.Constraint) (string, time.Time, error) {
	t := time.Now().UTC()
	promq := "cluster:health:components:map"

	if q.Id != "" {
		promq += fmt.Sprintf(`{group_id="%s"}`, q.Id)
	}

	if c != nil {
		if c.End != nil {
			t = *c.End
		}

		if c.Start != nil {
			duration := int(t.Sub(*c.Start).Seconds())
			if duration < 0 {
				return "", time.Time{}, fmt.Errorf("Start constraint is happening before the end time.")
			}
			promq = fmt.Sprintf("max_over_time(%s[%ds])", promq, duration)
		}
	}
	return promq, t, nil
}

func loadObjects(data model.Vector) []*Object {
	var incidents = make(map[string]*Object)
	for _, s := range data {
		labels := make(map[string]string)
		for k, v := range s.Metric {
			labels[string(k)] = string(v)
		}

		id := labels["group_id"]

		i, found := incidents[id]
		if !found {
			i = &Object{Id: id}
			incidents[id] = i
		}

		srcLabels := make(map[string]string)
		for k, v := range labels {
			if strings.HasPrefix(k, srcLabelPrefix) {
				srcLabels[k[len(srcLabelPrefix):]] = v
			}
		}
		if labels["type"] == "alert" {
			i.AlertsLabels = append(i.AlertsLabels, srcLabels)
		}
	}

	ret := make([]*Object, 0, len(incidents))
	for _, i := range incidents {
		ret = append(ret, i)
	}
	return ret
}

func filterObjects(objects []*Object, q Query) (ret []*Object) {
	for _, o := range objects {
		if len(q.AlertLabels) > 0 {
			// Check for any source labels matching the labels in the query.
			for _, l := range o.AlertsLabels {
				if isSubsetOf(l, q.AlertLabels) {
					log.V(3).Info("Incident matched alerts filter", "incident_id", o.Id)
					ret = append(ret, o)
					break
				}
			}
		} else {
			// No AlertLabels provided: no futher filtering needed.
			ret = append(ret, o)
		}
	}
	return ret
}

func isSubsetOf(part, whole map[string]string) bool {
	for k, v := range part {
		if whole[k] != v {
			return false
		}
	}
	return true
}
