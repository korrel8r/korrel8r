package rewrite

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func setup(t *testing.T) *Rewriter {
	t.Parallel()
	test.SkipIfNoCluster(t)
	e := engine.New()
	// TODO test with more domains.
	e.AddDomain(k8s.Domain, must.Must1(k8s.NewStore(test.K8sClient, test.RESTConfig)))
	e.AddDomain(loki.Domain, nil)
	u, err := openshift.ConsoleURL(context.Background(), test.K8sClient)
	require.NoError(t, err)
	return New(u, e)
}

func TestRewriter(t *testing.T) {
	rewriter := setup(t)
	for _, x := range []struct {
		cref, ref uri.Reference
		class     korrel8.Class
	}{
		// k8s refs
		{uri.Make("k8s/ns/default/pods/foo"), uri.Make("api/v1/namespaces/default/pods/foo"), k8s.ClassOf(&corev1.Pod{})},
		{uri.Make("k8s/ns/default/pods"), uri.Make("api/v1/namespaces/default/pods"), k8s.ClassOf(&corev1.Pod{})},
		{uri.Make("k8s/cluster/namespaces/default"), uri.Make("api/v1/namespaces/default"), k8s.ClassOf(&corev1.Namespace{})},
		{uri.Make("k8s/ns/openshift-logging/deployments"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments"), k8s.ClassOf(&appv1.Deployment{})},
		{uri.Make("k8s/ns/openshift-logging/deployments/foo"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments/foo"), k8s.ClassOf(&appv1.Deployment{})},
		// Loki refs
		{uri.Make("k8s/ns/openshift-logging/deployments/foo"), uri.Make("apis/apps/v1/namespaces/openshift-logging/deployments/foo"), k8s.ClassOf(&appv1.Deployment{})},
	} {
		t.Run("RefConsoleToStore/"+x.cref.Path, func(t *testing.T) {
			class, ref, err := rewriter.RefConsoleToStore(x.cref)
			if assert.NoError(t, err) {
				assert.Equal(t, x.class, class)
				assert.Equal(t, x.ref, ref)
			}
		})
		t.Run("RefStoreToConsole/"+x.ref.Path, func(t *testing.T) {
			cref, err := rewriter.RefStoreToConsole(x.class, x.ref)
			if assert.NoError(t, err) {
				assert.Equal(t, x.cref, cref)
			}
		})
	}
}
