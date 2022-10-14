package alert

import (
	"context"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var ctx = context.Background()

func TestOpenshiftHostFromRoute(t *testing.T) {
	c := fake.NewClientBuilder().Build()

	assert.NoError(t, c.Create(ctx, &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thanos-querier",
			Namespace: "openshift-monitoring",
		},
		Spec: routev1.RouteSpec{
			Host: "thanos.monitoring.example.com",
		},
	}))

	host, err := openshiftHostFromRoute(c)
	assert.NoError(t, err)
	assert.Equal(t, "thanos.monitoring.example.com", host)
}

func TestOpenshiftHostFromInClusterService(t *testing.T) {
	c := fake.NewClientBuilder().Build()

	assert.NoError(t, c.Create(ctx, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thanos-querier",
			Namespace: "openshift-monitoring",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{Name: "web", Port: 9091}},
		},
	}))

	host, err := openshiftHostFromInClusterService(c)
	assert.NoError(t, err)
	assert.Equal(t, "thanos-querier.openshift-monitoring:9091", host)
}
