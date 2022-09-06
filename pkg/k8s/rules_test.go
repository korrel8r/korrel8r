package k8s

import (
	"testing"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRules_DeploymentHasPods(t *testing.T) {
	r := korrel8.NewRuleGraph(Rules...).RulesWithStartAndGoal(ClassOf(&v1.Deployment{}), PodClass)[0]
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
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "foo"}},
				}},
			query: "/api/v1/namespaces/y/pods?labelSelector=app%3Dfoo",
		},
	} {
		t.Run(x.query, func(t *testing.T) {
			q, err := r.Follow(x.deployment)
			require.NoError(t, err)
			assert.Equal(t, x.query, string(q))
		})
	}
}
