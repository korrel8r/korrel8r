// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package openshift

import (
	"context"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	e, err := engine.Build().
		Domains(k8s.Domain, log.Domain, netflow.Domain, alert.Domain, metric.Domain).
		ConfigFile("../../etc/korrel8r/openshift-route.yaml").
		Engine()
	require.NoError(t, err)
	for _, q := range []string{
		`k8s:Pod:{}`,
		`log:infrastructure:{kubernetes_namespace_name=~".+"}`,
		`metric:metric:{namespace="kube-system"}`,
		`alert:alert:{severity: warning}`,
		`netflow:network:{DstK8S_Namespace=~".+"}`,
	} {
		t.Run(q, func(t *testing.T) {
			query, err := e.Query(q)
			limit := 3
			start := time.Now().Add(-time.Minute)
			constraint := &korrel8r.Constraint{Limit: &limit, Start: &start}
			r := korrel8r.NewResult(query.Class())
			if assert.NoError(t, err) {
				if assert.NoError(t, e.Get(context.Background(), query, constraint, r)) {
					// FIXME stores for metric and alert domains ignore the limit. Need to fix.
					assert.Equal(t, limit, len(r.List()), q)
				}
			}
		})
	}
}
