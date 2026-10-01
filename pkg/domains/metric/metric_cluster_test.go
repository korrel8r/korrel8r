// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/result"
	"github.com/korrel8r/korrel8r/pkg/rules/quickrules"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// metricRequireCluster returns a cluster client with the OpenShift Route type registered.
func metricRequireCluster(t testing.TB) client.Client {
	t.Helper()
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, routev1.AddToScheme(scheme))
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)
	return c
}

// ensureNamespacedRoute creates the namespace-scoped route if it is missing.
func ensureNamespacedRoute(t testing.TB, c client.Client) {
	t.Helper()
	key := types.NamespacedName{Namespace: "openshift-monitoring", Name: "thanos-querier-namespaced"}
	route := &routev1.Route{}
	err := c.Get(t.Context(), key, route)
	if err == nil {
		return
	}
	require.True(t, apierrors.IsNotFound(err), "get namespaced Thanos route: %v", err)

	route = &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		Spec: routev1.RouteSpec{
			To:   routev1.RouteTargetReference{Kind: "Service", Name: "thanos-querier"},
			Port: &routev1.RoutePort{TargetPort: intstr.FromInt(9092)},
			TLS:  &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt, InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect},
		},
	}
	err = c.Create(t.Context(), route)
	require.True(t, err == nil || apierrors.IsAlreadyExists(err), "create namespaced Thanos route: %v", err)
}

// clusterEngine returns an engine with all domains backed by an OpenShift cluster,
// configured as in the CLI: configuration files plus the compiled quickrules.
func clusterEngine(t testing.TB) *engine.Engine {
	t.Helper()
	out, err := exec.Command("git", "root").Output()
	require.NoError(t, err)
	configFile := filepath.Join(strings.TrimSpace(string(out)), "etc", "korrel8r", "openshift-route.yaml")
	b := engine.Build().Domains(domains.All...)
	e, err := b.ConfigFile(configFile).
		Rules(quickrules.Rules(b.GetDomains())...).
		StatusRules(quickrules.StatusRules(b.GetDomains())...).
		Engine()
	require.NoError(t, err)
	return e
}

// getMetrics runs query against the engine's metric store and returns the series.
func getMetrics(t testing.TB, e *engine.Engine, query string) []metric.Object {
	t.Helper()
	q, err := e.Query(query)
	require.NoError(t, err, query)
	r := result.New(q.Class())
	require.NoError(t, e.Get(t.Context(), q, nil, r), query)
	objects := r.List()
	series := make([]metric.Object, 0, len(objects))
	for _, o := range objects {
		series = append(series, o.(metric.Object))
	}
	return series
}

// TestMetricNamespaceSuffix_cluster tests namespace suffix queries against a real
// Prometheus/Thanos store: parse-time validation, explicit namespace scoping,
// and queries without a namespace suffix.
func TestMetricNamespaceSuffix_cluster(t *testing.T) {
	c := metricRequireCluster(t)
	ensureNamespacedRoute(t, c)
	e := clusterEngine(t)

	t.Run("parse", func(t *testing.T) {
		for _, s := range []string{
			`metric:metric:kube_pod_info?namespace=kube-system`,
			`metric:metric:kube_pod_info?namespace=a&namespace=b`,
		} {
			q, err := metric.Domain.Query(s)
			require.NoError(t, err, s)
			assert.Equal(t, s, q.String())
			q2, err := metric.Domain.Query(q.String()) // Round trip.
			require.NoError(t, err, s)
			assert.Equal(t, q, q2)
		}
		for _, s := range []string{
			`metric:metric:kube_pod_info?tenant=x`,
			`metric:metric:kube_pod_info?namespace=`,
			`metric:metric:kube_pod_info?a=%zz`,
		} {
			_, err := metric.Domain.Query(s)
			require.Error(t, err, s)
		}
	})

	t.Run("explicit", func(t *testing.T) {
		series := getMetrics(t, e, `metric:metric:{__name__!=""}?namespace=kube-system`)
		require.NotEmpty(t, series)
		for _, s := range series {
			assert.Equal(t, "kube-system", s.Labels["namespace"], "series not scoped to suffix namespace: %v", s)
		}
	})

	t.Run("explicitWithoutMatcher", func(t *testing.T) {
		namespace := test.TempNamespace(t, c, "metricsuffix-").Name
		// Empty results are acceptable, errors are not.
		getMetrics(t, e, `metric:metric:kube_pod_info?namespace=`+namespace)
	})

	t.Run("withoutSuffix", func(t *testing.T) {
		// The unsuffixed query uses the cluster-wide URL, with no namespace parameters.
		getMetrics(t, e, `metric:metric:kube_pod_info{namespace="kube-system"}`)
	})
}

// TestMetricRules_cluster tests the updated compiled rules end-to-end: apply
// AllToMetric to a real cluster object and run the generated suffixed query.
func TestMetricRules_cluster(t *testing.T) {
	c := metricRequireCluster(t)
	ensureNamespacedRoute(t, c)
	e := clusterEngine(t)
	namespace := test.TempNamespace(t, c, "metricrule-").Name

	pod := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      "metric-pod",
			"namespace": namespace,
		},
		"spec": map[string]any{
			"containers": []any{map[string]any{"name": "busybox", "image": "busybox"}},
		},
	}}
	require.NoError(t, c.Create(t.Context(), pod))

	r := e.Rule("AllToMetric")
	require.NotNil(t, r, "missing rule AllToMetric")
	queries, err := r.Apply(k8s.FromUnstructured(pod))
	require.NoError(t, err)
	require.Len(t, queries, 1)

	var q = queries[0]
	assert.Equal(t, fmt.Sprintf(`metric:metric:{namespace="%v",pod="metric-pod"}?namespace=%v`, namespace, namespace), q.String())

	// The generated query must be accepted and must not fail against the store.
	// Empty results are acceptable, errors are not.
	resultObj := result.New(q.Class())
	require.NoError(t, e.Get(t.Context(), q, nil, resultObj))
}
