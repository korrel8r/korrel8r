package rules

import (
	"testing"

	"github.com/alanconway/korrel8/pkg/alert"
	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRules_DeploymentHasPods(t *testing.T) {
	rs := korrel8.NewRuleSet()
	AddTo(rs)
	rules := rs.GetRules(k8s.ClassOf(&appsv1.Deployment{}), k8s.ClassOf(&corev1.Pod{}))
	require.NotEmpty(t, rules)
	r := rules[0]
	for _, x := range []struct {
		deployment *appsv1.Deployment
		query      string
	}{
		{
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "x"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "foo"}},
				}},
			query: "/api/v1/namespaces/x/pods?labelSelector=app%3Dfoo",
		},
		{
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "y"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "foo", "x": "y"}},
				}},
			query: "/api/v1/namespaces/y/pods?labelSelector=app%3Dfoo%2Cx%3Dy",
		},
	} {
		t.Run(x.query, func(t *testing.T) {
			result, err := r.Apply(k8s.Object{Object: x.deployment}, nil)
			require.NoError(t, err)
			assert.Len(t, result, 1)
			assert.Equal(t, x.query, result[0])
		})
	}
}

func TestRules_ALertToK8s(t *testing.T) {
	a := alert.Object{GettableAlert: &models.GettableAlert{
		Alert: models.Alert{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}}},
	}
	r := All().GetRules(alert.Class{}, k8s.ClassOf(&v1.Deployment{}))
	assert.NotEmpty(t, r)
	q, err := r[0].Apply(a, nil)
	assert.NoError(t, err)
	assert.Equal(t, korrel8.Queries{"/api/v1/namespaces/foo/deployments/bar"}, q)
}
