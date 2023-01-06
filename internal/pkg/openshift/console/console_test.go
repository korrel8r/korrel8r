package console

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func setup(t *testing.T) *Console {
	t.Parallel()
	test.SkipIfNoCluster(t)
	e := engine.New()
	// TODO test with more domains.
	e.AddDomain(k8s.Domain, k8s.NewStore(test.K8sClient))
	u, err := openshift.ConsoleURL(context.Background(), test.K8sClient)
	require.NoError(t, err)
	return New(u, e)
}

func TestConsoleToRef(t *testing.T) {
	// FIXME add domains
	console := setup(t)
	for _, x := range []struct {
		cref, ref uri.Reference
		class     korrel8.Class
	}{
		{uri.Make("/k8s/ns/default/pods/foo"), uri.Make("api/v1/namespaces/default/pods/foo"), k8s.ClassOf(&corev1.Pod{})},
		{uri.Make("/k8s/ns/default/pods"), uri.Make("api/v1/namespaces/default/pods"), k8s.ClassOf(&corev1.Pod{})},
		{uri.Make("/k8s/cluster/projects/default"), uri.Make("api/v1/namespaces/default"), k8s.ClassOf(&corev1.Namespace{})},
		{uri.Make("/k8s/cluster/namespaces/default"), uri.Make("api/v1/namespaces/default"), k8s.ClassOf(&corev1.Namespace{})},
		{uri.Make("/k8s/ns/openshift-logging/deployments"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments"), k8s.ClassOf(&appv1.Deployment{})},
		{uri.Make("/k8s/ns/openshift-logging/deployments/foo"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments/foo"), k8s.ClassOf(&appv1.Deployment{})},
	} {
		t.Run(x.cref.Path, func(t *testing.T) {
			// FIXME complete this test.
			class, ref, err := console.ConsoleToRef(x.cref)
			if assert.NoError(t, err) {
				assert.Equal(t, x.class, class)
				assert.Equal(t, x.ref, ref)
			}
		})
	}
}

func TestRefToConsole(t *testing.T) {
	console := setup(t)
	for _, x := range []struct {
		cref, ref uri.Reference
		class     korrel8.Class
	}{
		{uri.Make("k8s/ns/default/pods/foo"), uri.Make("api/v1/namespaces/default/pods/foo"), k8s.ClassOf(&corev1.Pod{})},
		{uri.Make("k8s/ns/default/pods"), uri.Make("api/v1/namespaces/default/pods"), k8s.ClassOf(&corev1.Pod{})},
		{uri.Make("k8s/cluster/namespaces/default"), uri.Make("api/v1/namespaces/default"), k8s.ClassOf(&corev1.Namespace{})},
		{uri.Make("k8s/ns/openshift-logging/deployments"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments"), k8s.ClassOf(&appv1.Deployment{})},
		{uri.Make("k8s/ns/openshift-logging/deployments/foo"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments/foo"), k8s.ClassOf(&appv1.Deployment{})},
	} {
		t.Run(x.cref.Path, func(t *testing.T) {
			// FIXME complete this test.
			cref, err := console.RefToConsole(x.ref)
			if assert.NoError(t, err) {
				assert.Equal(t, x.cref, cref)
			}
		})
	}
}
